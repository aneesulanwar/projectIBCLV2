
package projectIBCLV2

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"

	chain "github.com/aneesulanwar/projectIBCV2"
)

type Node struct {
	//Node represent the person
	Name    string
	Address string
	Port    string
}

type CAddress struct {
	//CAddress to store addresses of connected nodes
	Name    string
	Address string
	Port    string
}

type NetworkTrans struct {
	Name        string
	Data        string
	Block       *chain.Block
	Bchain      *chain.Block
	Addresses   []CAddress
	Transaction chain.Transaction
}

var Distributed bool
var Nodes []CAddress
var MinimumNodes int

var Leader CAddress

var myStake float64

var validation_score = make(map[string]float64) //stores validation score for a node

var stake_amount = make(map[string]float64) //stores stake of a node

var trust_score = make(map[string]float64) //stores the calculated trust score of a node

func HandleConnection(con net.Conn, leader CAddress, chainHead **chain.Block) {

	var recvdBlock NetworkTrans
	dec := gob.NewDecoder(con)
	err := dec.Decode(&recvdBlock)
	if err != nil {
		// handle error
	}

	if recvdBlock.Name == "Starting" {

		///storing the adrresses of new connected Node
		node := strings.Fields(string(recvdBlock.Data))
		fmt.Println(node[0], node[1], node[2])

		var nodes CAddress
		nodes.Name = node[0]
		nodes.Address = node[1]
		nodes.Port = node[2]
		find := false
		for i := 0; i < len(Nodes); i++ {
			if Nodes[i].Name == nodes.Name {
				find = true
			}
		}
		if find == false {
			Nodes = append(Nodes, nodes)
			con.Write([]byte("accepted" + " "))
			*(chainHead) = MineBlock(leader, *(chainHead))

			//storing validation and trust score and stake amount
			validation_score[nodes.Name] = 1.0
			stake_amount[nodes.Name] = 0.0
			trust_score[nodes.Name] = calculateTrustScore(nodes.Name)
			////////////////////////////////////////////////////////

			if Distributed == true {
				Distribute(*(chainHead), nodes, len(Nodes)-1)
				var newTran chain.Transaction
				newTran.To = leader.Name
				newTran.From = "mining"
				newTran.Bcoins = 100

				var newTran1 chain.Transaction
				newTran1.To = "Burn"
				newTran1.From = leader.Name
				newTran1.Bcoins = myStake

				var Block chain.Block
				Block.Transactions = append(Block.Transactions, newTran)
				Block.DeriveHash()
				Block1 := &Block
				/////
				for i := 0; i < len(Nodes); i++ {
					Propagate(newTran1, Block1, Nodes[i])
				}

			}

			if len(Nodes) >= MinimumNodes && Distributed == false {
				i := 0
				for i < len(Nodes)-1 {
					Distribute(*(chainHead), Nodes[i], len(Nodes)-1)
					i = i + 1
				}
				Distributed = true
				Distribute(*(chainHead), Nodes[len(Nodes)-1], len(Nodes)-1)
				chain.ListBlocks(*(chainHead))
			}
			//con.Write([]byte(leader.Address + " " + leader.Port + " "))

			//chain.ListBlocks(*(chainHead))
		} else {
			con.Write([]byte("nameAlreadyExists" + " "))
		}
		////////////////////////////////////////////////////
		//mining block on addition of new node

	}

	if recvdBlock.Name == "Validate" {
		Validate(recvdBlock.Transaction, leader, chainHead)
	}

	if recvdBlock.Name == "ValidateBlock" {
		ValidateBlock(leader, recvdBlock.Transaction, recvdBlock.Block, chainHead)
	}

	if recvdBlock.Name == "Stake" {
		validateStake(recvdBlock.Transaction, chainHead)
	}

}

func validateStake(trans chain.Transaction, chainHead **chain.Block) {
	tempv := *(chainHead)
	amount := 0.0
	for tempv.PrevPointer != nil {
		i := 0
		for i < len(tempv.Transactions) {
			if tempv.Transactions[i].To == trans.From {
				amount += tempv.Transactions[i].Bcoins
			}
			if tempv.Transactions[i].From == trans.From {
				amount -= tempv.Transactions[i].Bcoins
			}
			i = i + 1
		}
		tempv = tempv.PrevPointer
	}
	i := 0
	for i < len(tempv.Transactions) {
		if tempv.Transactions[i].To == trans.From {
			amount += tempv.Transactions[i].Bcoins
		}
		if tempv.Transactions[i].From == trans.From {
			amount -= tempv.Transactions[i].Bcoins
		}
		i = i + 1
	}

	if amount >= trans.Bcoins {
		stake_amount[trans.From] = trans.Bcoins
		trust_score[trans.From] = calculateTrustScore(trans.From)
	}

}

func chooseValidator() int {
	max := findMax(trust_score)
	rangee := max
	if max > 1.5 {
		rangee--
	} else if rangee > 0.5 && rangee < 1.5 {
		rangee -= 0.5
	}
	nodes := retMaxNodes(rangee)
	fmt.Println("max ", max)
	fmt.Println(nodes)
	lenNodes := len(nodes)
	validator := rand.Intn(lenNodes)

	retValidator := 0
	for i := 0; i < len(Nodes); i++ {
		if Nodes[i].Name == nodes[validator] {
			retValidator = i
		}
	}

	return retValidator
}

func retMaxNodes(value float64) []string {
	adds := []string{}
	for k, v := range trust_score {
		if v >= value {
			adds = append(adds, k)
		}
	}
	return adds
}

func findMax(mapNodes map[string]float64) float64 {
	max := 0.0
	for _, v := range mapNodes {
		if v > max {
			max = v
		}
	}
	return max
}

func MineBlock(N CAddress, chainHead *chain.Block) *chain.Block {
	var newTrans chain.Transaction
	var transactions []chain.Transaction
	newTrans.From = "mining"
	newTrans.To = N.Name
	newTrans.Bcoins = 100
	transactions = append(transactions, newTrans)
	chainHead = chain.InsertBlock(transactions, chainHead)
	return chainHead
}

func Distribute(chainHead *chain.Block, node CAddress, index int) {

	conn, err := net.Dial("tcp", node.Address+":"+node.Port)
	if err != nil {
		// handle error
		log.Println(err)
		fmt.Println("error in connection")

	}

	var blck NetworkTrans
	blck.Name = "FirstUpdate"
	blck.Bchain = chainHead

	var indexes []int
	j := 0
	for j < index {
		ind := rand.Intn(len(Nodes))
		if len(indexes) == 0 {
			if Nodes[ind].Name != node.Name {
				indexes = append(indexes, ind)
				j = j + 1
			}
		}
		if len(indexes) > 0 {
			if IsPresent(indexes, ind) == false {
				if Nodes[ind].Name != node.Name {
					indexes = append(indexes, ind)
					j = j + 1
				}
			}
		}
	}

	for k := 0; k < len(indexes); k++ {
		blck.Addresses = append(blck.Addresses, Nodes[indexes[k]])
	}
	gobEncoder := gob.NewEncoder(conn)
	err1 := gobEncoder.Encode(blck)
	if err1 != nil {
		log.Println(err)
	}
}

func IsPresent(list []int, number int) bool {
	found := false
	for i := 0; i < len(list); i++ {
		if list[i] == number {
			found = true
			break
		}
	}

	return found
}
func WantTransaction(beginer CAddress, chainHead **chain.Block) {
	for {
		if Distributed == true {
			fmt.Println("do you want to perform transaction?")
			var trans string
			fmt.Scan(&trans)
			if trans == "yes" {
				var wg sync.WaitGroup
				wg.Add(1)
				StartTransaction(beginer, chainHead, &wg)
				wg.Wait()
			}
		}
	}
}

func sendTransaction(newTrans chain.Transaction, node CAddress) {
	var newBlock NetworkTrans

	if newTrans.To == "stake" {
		newBlock.Name = "Stake"
	} else {
		newBlock.Name = "Validate"
	}
	newBlock.Transaction = newTrans

	validator := chooseValidator()
	max := findMax(trust_score)
	validNode := Nodes[validator]
	if max <= 0 {
		validNode = node
	}
	if newTrans.To != "stake" {

		conn, err := net.Dial("tcp", validNode.Address+":"+validNode.Port)
		if err != nil {
			// handle error
			log.Println(err)
			fmt.Println("error in connection")

		}
		gobEncoder := gob.NewEncoder(conn)
		err1 := gobEncoder.Encode(newBlock)
		if err1 != nil {
			log.Println(err)
		}
	} else {
		for i := 0; i < len(Nodes); i++ {
			conn, err := net.Dial("tcp", Nodes[i].Address+":"+Nodes[i].Port)
			if err != nil {
				// handle error
				log.Println(err)
				fmt.Println("error in connection")

			}
			gobEncoder := gob.NewEncoder(conn)
			err1 := gobEncoder.Encode(newBlock)
			if err1 != nil {
				log.Println(err)
			}
		}
	}
}

func StartTransaction(beginer CAddress, chainHead **chain.Block, wg *sync.WaitGroup) {
	fmt.Println("enter the name of receiver")
	var receiver string
	fmt.Scan(&receiver)

	fmt.Println("enter the amount of Bcoins you want to transfer")
	var amount float64
	fmt.Scan(&amount)

	var newTrans chain.Transaction
	newTrans.To = receiver
	newTrans.From = beginer.Name
	newTrans.Bcoins = amount

	var newBlock NetworkTrans
	if newTrans.To == "stake" {
		for amount > 100 {
			fmt.Println("enter the amount of stake again, can't have stake greater than 100")
			fmt.Scan(&amount)
		}
		newTrans.Bcoins = amount
		myStake = amount
	} else {
		newBlock.Name = "Validate"
	}
	sendTransaction(newTrans, beginer)

	defer wg.Done()
}

func ValidateTransaction(transaction chain.Transaction, node CAddress, chainHead **chain.Block) {

	fmt.Println("received validate from client")
	validator := chooseValidator()
	max := findMax(trust_score)
	send := true
	if max <= 0 { // if no body has coins on stake......then server will mine the block
		if transaction.To == "Burn" {
			if stake_amount[transaction.From] == 0 {
				send = false
			} else {
				stake_amount[transaction.From] = 0
				trust_score[transaction.From] = calculateTrustScore(transaction.From)
			}
		}
		if send {
			Validate(transaction, node, chainHead)
		}
	} else {
		var newBlock NetworkTrans
		newBlock.Name = "Validate"
		newBlock.Transaction = transaction

		fmt.Println("Validator is ", Nodes[validator].Name)
		if transaction.To == "Burn" {
			if stake_amount[transaction.From] == 0 {
				send = false
			} else {
				stake_amount[transaction.From] = 0
				trust_score[transaction.From] = calculateTrustScore(transaction.From)
			}
		}
		if send {
			conn, err := net.Dial("tcp", Nodes[validator].Address+":"+Nodes[validator].Port)
			if err != nil {
				// handle error
				log.Println(err)
				fmt.Println("error in connection")
			}
			gobEncoder := gob.NewEncoder(conn)
			err1 := gobEncoder.Encode(newBlock)
			if err1 != nil {
				log.Println(err)
			}
		}

	}

}

func vlaidateStake(transaction chain.Transaction, chainHead **chain.Block) bool {
	var temp *chain.Block
	temp = *(chainHead)
	amount := 0.0
	res := false
	for temp.PrevPointer != nil {
		i := 0
		for i < len(temp.Transactions) {
			if temp.Transactions[i].To == transaction.From {
				amount += temp.Transactions[i].Bcoins
			}
			if temp.Transactions[i].From == transaction.From {
				amount -= temp.Transactions[i].Bcoins
			}
			i = i + 1
		}
		temp = temp.PrevPointer
	}
	i := 0
	for i < len(temp.Transactions) {
		if temp.Transactions[i].To == transaction.From {
			amount += temp.Transactions[i].Bcoins
		}
		if temp.Transactions[i].From == transaction.From {
			amount -= temp.Transactions[i].Bcoins
		}
		i = i + 1
	}

	if amount < transaction.Bcoins {
		res = false
	}
	if amount >= transaction.Bcoins {
		res = true
	}

	return res

}

func Validate(transaction chain.Transaction, thisNode CAddress, chainHead **chain.Block) {

	fmt.Println("received Validate Transaction")

	validTransaction := true
	toSend := true
	var temp *chain.Block
	temp = *(chainHead)
	amount := 0.0
	for temp.PrevPointer != nil {
		i := 0
		for i < len(temp.Transactions) {
			if temp.Transactions[i].To == transaction.From {
				amount += temp.Transactions[i].Bcoins
			}

			if temp.Transactions[i].From == transaction.From {
				amount -= temp.Transactions[i].Bcoins
			}
			i = i + 1
		}
		temp = temp.PrevPointer
	}
	i := 0
	for i < len(temp.Transactions) {
		if temp.Transactions[i].To == transaction.From {
			amount += temp.Transactions[i].Bcoins
		}
		if temp.Transactions[i].From == transaction.From {
			amount -= temp.Transactions[i].Bcoins
		}
		i = i + 1
	}

	if amount < transaction.Bcoins {
		fmt.Println("Invalid Transaction")
		decision := rand.Intn(4)
		if decision == 0 {
			validTransaction = true
		} else {
			validTransaction = false
		}

		fmt.Println("Decision is ", validTransaction)

	}

	if transaction.To == "Burn" {
		if stake_amount[transaction.From] == 0 {
			toSend = false
		} else {
			stake_amount[transaction.From] = 0
			toSend = true
		}
	}

	if validTransaction && toSend {
		fmt.Println("Valid Transaction")
		var newTran chain.Transaction
		newTran.To = thisNode.Name
		newTran.From = "mining"
		newTran.Bcoins = 100

		var newTran1 chain.Transaction
		newTran1.To = "Burn"
		newTran1.From = thisNode.Name
		newTran1.Bcoins = myStake

		var Block chain.Block
		Block.Transactions = append(Block.Transactions, newTran)
		Block.Transactions = append(Block.Transactions, transaction)
		Block.DeriveHash()
		toAdd := &Block
		Block1 := &Block
		/////
		temp := *(chainHead)
		toAdd.PrevBlockHash = temp.Hash
		toAdd.PrevPointer = temp
		ValidateBlock(thisNode, newTran1, toAdd, chainHead)
		Block1.PrevPointer = temp
		Block1.PrevBlockHash = temp.Hash
		//*(chainHead) = toAdd
		/////
		for i := 0; i < len(Nodes); i++ {
			Propagate(newTran1, Block1, Nodes[i])
		}

		chain.ListBlocks(*(chainHead))
	} else if toSend {
		fmt.Println("InValid Transaction")
		var newTran chain.Transaction
		newTran.To = thisNode.Name
		newTran.From = "mining"
		newTran.Bcoins = 100

		var newTran1 chain.Transaction
		newTran1.To = "Burn"
		newTran1.From = thisNode.Name
		newTran1.Bcoins = myStake

		var Block chain.Block
		Block.Transactions = append(Block.Transactions, newTran)
		Block.DeriveHash()
		toAdd := &Block
		Block1 := &Block
		/////
		temp := *(chainHead)
		toAdd.PrevBlockHash = temp.Hash
		toAdd.PrevPointer = temp
		ValidateBlock(thisNode, newTran1, toAdd, chainHead)
		Block1.PrevPointer = temp
		Block1.PrevBlockHash = temp.Hash
		//*(chainHead) = toAdd
		///// //////////////////////////////////////////////////////////////
		for i := 0; i < len(Nodes); i++ {
			Propagate(newTran1, Block1, Nodes[i])
		}

		chain.ListBlocks(*(chainHead))
	}

}

func Propagate(trans chain.Transaction, block *chain.Block, node CAddress) {
	conn, err := net.Dial("tcp", node.Address+":"+node.Port)
	if err != nil {
		// handle error
		log.Println(err)
		fmt.Println("error in connection")

	}

	var blck NetworkTrans
	blck.Name = "ValidateBlock"
	blck.Block = block
	blck.Transaction = trans
	gobEncoder := gob.NewEncoder(conn)
	err1 := gobEncoder.Encode(blck)
	if err1 != nil {
		log.Println(err)
	}
}

func ValidateBlock(node CAddress, trans chain.Transaction, block *chain.Block, chainHead **chain.Block) {

	//fmt.Println("received Validate Block")

	validb := true //if block is valid
	var minor string
	var tempv *chain.Block
	tempv = *(chainHead)
	tempb := block
	stakevalid := true
	for t := 0; t < len(tempb.Transactions); t++ {
		amount := 0.0
		if tempb.Transactions[t].From != "mining" {
			if tempb.Transactions[t].To == "Burn" {
				stake_amount[tempb.Transactions[t].From] = 0
			}
			for tempv.PrevPointer != nil {
				i := 0
				for i < len(tempv.Transactions) {
					if tempv.Transactions[i].To == tempb.Transactions[t].From {
						amount += tempv.Transactions[i].Bcoins
					}
					if tempv.Transactions[i].From == tempb.Transactions[t].From {
						amount -= tempv.Transactions[i].Bcoins
					}
					i = i + 1
				}
				tempv = tempv.PrevPointer
			}
			i := 0
			for i < len(tempv.Transactions) {
				if tempv.Transactions[i].To == tempb.Transactions[t].From {
					amount += tempv.Transactions[i].Bcoins
				}
				if tempv.Transactions[i].From == tempb.Transactions[t].From {
					amount -= tempv.Transactions[i].Bcoins
				}
				i = i + 1
			}

			if amount < tempb.Transactions[t].Bcoins {
				validb = false

			}
		} else if tempb.Transactions[t].From == "mining" {
			minor = tempb.Transactions[t].To
			if tempb.Transactions[t].Bcoins != 100 {
				validb = false
			}
		}

		tempv = *(chainHead)
	}

	amount := 0.0
	for tempv.PrevPointer != nil {
		i := 0
		for i < len(tempv.Transactions) {
			if tempv.Transactions[i].To == trans.From {
				amount += tempv.Transactions[i].Bcoins
			}
			if tempv.Transactions[i].From == trans.From {
				amount -= tempv.Transactions[i].Bcoins
			}
			i = i + 1
		}
		tempv = tempv.PrevPointer
	}
	i := 0
	for i < len(tempv.Transactions) {
		if tempv.Transactions[i].To == trans.From {
			amount += tempv.Transactions[i].Bcoins
		}
		if tempv.Transactions[i].From == trans.From {
			amount -= tempv.Transactions[i].Bcoins
		}
		i = i + 1
	}

	if amount < trans.Bcoins && trans.Bcoins == stake_amount[minor] {
		stakevalid = false
	}

	if validb && stakevalid {
		temp1 := block
		temp := *(chainHead)
		result := bytes.Compare(block.Hash, temp.Hash)
		result1 := bytes.Compare(block.PrevBlockHash, temp.PrevBlockHash)
		len1 := length(block)
		len2 := length(*(chainHead))
		if result != 0 || result1 != 0 || len1 > len2 {
			block.PrevBlockHash = temp.Hash
			block.PrevPointer = temp
			*(chainHead) = block
			i := 0
			for i < len(Nodes) {
				Propagate(trans, temp1, Nodes[i])
				i = i + 1
			}

			chain.ListBlocks(*(chainHead))
		}
	} else if validb == false && stakevalid == true {
		sendTransaction(trans, node)
	} else if stakevalid == false {
		stake_amount[minor] = 0
		trust_score[minor] = 0
		validation_score[minor] = validation_score[minor] / 2.0
	}
}

func length(block *chain.Block) int {
	temp := block
	len := 0
	for temp != nil {
		len++
		temp = temp.PrevPointer
	}

	return len
}

func calculateTrustScore(name string) float64 {
	validationscore := validation_score[name]
	stakeamount := stake_amount[name]
	trustvalue := 0.0
	if stakeamount == 0 {
		trustvalue = 0
	} else {
		tempstake := stakeamount / 100.0
		temptrust := validationscore * 3
		trustvalue = tempstake + temptrust

	}

	return trustvalue
}
