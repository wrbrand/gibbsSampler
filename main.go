package main

import (
	"os"
	"bufio"
	"log"
	"strings"
	"math/rand"
	"fmt"
)

type Transaction struct {
	sender, recipient, amount string
}

type SenderAmount struct {
	recipient, amount string
}

func generateDependentDistributions(transactions []Transaction) (map[string]map[string]float64, map[string]map[string]float64, map[string]map[string]float64) {
	// We now need a variety of distributions of one of (sender, recipient, amount) based on a given value for the other two
	// map of (sender, recipient) -> amount
	// map of (sender, amount) -> recipient
	// map of (recipient, amount) -> sender

	var recipientCountGivenSenderAmount = make(map[string]map[string]int)
	var senderCountGivenRecipientAmount = make(map[string]map[string]int)
	var amountCountGivenSenderRecipient = make(map[string]map[string]int)

	for _, t := range transactions {
		if(t.sender == "" || t.recipient == "" || t.amount == "") {
			continue
		}

		incrementOrCreateCount(t.sender, t.amount, recipientCountGivenSenderAmount, t.recipient)
		incrementOrCreateCount(t.recipient, t.amount, senderCountGivenRecipientAmount, t.sender)
		incrementOrCreateCount(t.sender, t.recipient, amountCountGivenSenderRecipient, t.amount)
	}

	// Our aCountGivenBC variables are now maps to amounts; we need to remap them to probabilities
	var recipientProbabilitiesGivenSenderAmount = countsToProbabilities(recipientCountGivenSenderAmount)
	var senderProbabilitiesGivenRecipientAmount = countsToProbabilities(senderCountGivenRecipientAmount)
	var amountProbabilitiesGivenSenderRecipient = countsToProbabilities(amountCountGivenSenderRecipient)

	return recipientProbabilitiesGivenSenderAmount, senderProbabilitiesGivenRecipientAmount, amountProbabilitiesGivenSenderRecipient
}

func countsToProbabilities(countsMap map[string]map[string]int) (map[string]map[string]float64){
	var probabilitiesMap = make(map[string]map[string]float64)

	for outerKey, innerMap := range countsMap {
		total := 0
		probabilitiesMap[outerKey] = make(map[string]float64)

		for _, count := range innerMap {
			total += count
		}

		for innerKey, count := range innerMap {
			probabilitiesMap[outerKey][innerKey] = float64(count) / float64(total)
		}
	}

	return probabilitiesMap
}

func incrementOrCreateCount(keyStart string, keyEnd string, outerMap map[string]map[string]int, innerKey string) {
	outerKey := strings.Join([]string{ keyStart, keyEnd }, "")
	innerMap, ok := outerMap[outerKey]

	if ok {
		_, okk := innerMap[innerKey]
		if okk {
			innerMap[innerKey] += 1
		} else {
			innerMap[innerKey] = 1
		}
	} else {
		outerMap[outerKey] = map[string]int { innerKey: 1 }
	}
}

func sample(probabilities map[string]float64) (string) {
	cumProbabilities := make(map[string]float64)
	sum := 0.0

	// Create an array of keys to iterate over, since iteration order of maps is not guaranteed
	keys := make([]string, len(probabilities))
	for k, _ := range probabilities {
		keys = append(keys, k)
	}

	for _, key := range keys {
		cumProbabilities[key] = sum + probabilities[key]
		sum += probabilities[key]
	}

	random := rand.Float64()

	for _, key := range keys {
		if random <= cumProbabilities[key] {
			return key
		}
	}

	log.Fatal("Failed to sample")

	return ""
}

func findLargestConnectedSubgraph(transactions []Transaction) ([]Transaction) {
	// graphs maps connected sender/recipient addresses to numbered subgraphs, starting at 1
	graphs := make(map[string]int)
	nextGraph := 1

	for _, t := range transactions {
		if graphs[t.sender] != 0 && graphs[t.recipient] != 0 {
			graphs[t.recipient] = graphs[t.sender]
		} else if graphs[t.sender] == 0 && graphs[t.recipient] != 0 {
			graphs[t.sender] = graphs[t.recipient]
		} else if graphs[t.sender] == 0 && graphs[t.recipient] == 0 {
			graphs[t.sender] = nextGraph
			graphs[t.recipient] = nextGraph
			nextGraph++
		}
	}

	largestGraphNodes := make(map[string]bool)
	largestGraphSize := 0

	for i := 1; i < nextGraph; i++ {
		graphSize := 0
		nodes := make(map[string]bool)

		for node, graph := range graphs {
			if graph == i {
				nodes[node] = true
				graphSize++
			}
		}

		if graphSize > largestGraphSize {
			largestGraphSize = graphSize
			largestGraphNodes = nodes
		}
	}

	var subgraphTransactions []Transaction

	for _, t := range transactions {
		if largestGraphNodes[t.sender] || largestGraphNodes[t.recipient] {
			subgraphTransactions = append(subgraphTransactions, t)
		}
	}

	return subgraphTransactions
}

func main() {
	iterations := 5000

	rand.Seed(5)
	inputFile := os.Args[1]

	file, err := os.Open(inputFile)

	if err != nil {
		log.Fatal(err)
	}

	var transactions []Transaction
	var txLen = 0
	scanner := bufio.NewScanner(file)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ",")

		// Ignore first field, assuming it's a block ID
		transactions = append(transactions, Transaction{fields[1], fields[2], fields[3]})
		txLen++
	}

	recipientProbabilitiesGivenSenderAmount, senderProbabilitiesGivenRecipientAmount, amountProbabilitiesGivenSenderRecipient := generateDependentDistributions(transactions)

	var sampledTransactions []Transaction
	subgraphTransactions := findLargestConnectedSubgraph(transactions)

	//fmt.Printf("Largest subgraph has %d transactions\n", len(subgraphTransactions))

	sampledTransactions = append(sampledTransactions, subgraphTransactions[rand.Intn(len(transactions) - 1)])

	for n := 1; n < iterations; n++ {
		sender := sample(senderProbabilitiesGivenRecipientAmount[strings.Join([]string { sampledTransactions[n-1].recipient, sampledTransactions[n-1].amount }, "")]);
		recipient := sample(recipientProbabilitiesGivenSenderAmount[strings.Join([]string { sender, sampledTransactions[n-1].amount }, "")]);
		amount := sample(amountProbabilitiesGivenSenderRecipient[strings.Join([]string { sender, recipient }, "")]);

		newTransaction := Transaction{sender, recipient, amount }

		/*if newTransaction == sampledTransactions[n - 1] {
			sampledTransactions = append(sampledTransactions, transactions[rand.Intn(len(transactions) - 1)])
		} else {
			sampledTransactions = append(sampledTransactions, Transaction{sender,recipient,amount})
		}*/

		sampledTransactions = append(sampledTransactions, newTransaction)
	}

	for _, i := range sampledTransactions {
		fmt.Printf("%s,%s,%s\n",i.sender, i.recipient, i.amount)
	}
}
