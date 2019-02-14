package invoices

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// BookedInvoices - endpoint https://restapi.e-conomic.com/invoices/booked
type BookedInvoices struct {
	Collection []*BookedInvoice `json:"collection,omitempty"`
	Pagination *Pagination      `json:"pagination,omitempty"`
}

// Pagination for getting more invoices
type Pagination struct {
	PageSize  int    `json:"pageSize,omitempty"`
	Results   int    `json:"results,omitempty"` //Total results
	FirstPage string `json:"firstPage,omitempty"`
	NextPage  string `json:"nextPage,omitempty"`
	LastPage  string `json:"lastPage,omitempty"`
}

// BookedInvoice - endpoint https://restapi.e-conomic.com/invoices/booked/:number
type BookedInvoice struct {
	BookedInvoiceNumber int        `json:"bookedInvoiceNumber,omitempty"`
	Date                string     `json:"date,omitempty"`
	Currency            string     `json:"currency,omitempty"`
	NetAmount           float64    `json:"netAmount,omitempty"`
	GrossAmount         float64    `json:"grossAmount,omitempty"`
	VatAmount           float64    `json:"vatAmount,omitempty"`
	Lines               []*Lines   `json:"lines,omitempty"`
	Recipient           *Recipient `json:"recipient"`
}

// Recipient -
type Recipient struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// Lines -
type Lines struct {
	LineNumber     byte    `json:"lineNumber,omitempty"`  /*MUST be #2 on voice*/
	Description    string  `json:"description,omitempty"` /*If this == dragonplan*/
	CreditQuantity float64 `json:"quantity,omitempty"`    /*Number of credits*/
}

var (
	logger *log.Logger
	ecoURL = "https://restapi.e-conomic.com"
)

const (
	creditLineNumber = 2
	photographerCut  = 15
)

/**
 * 1. Get all invoiceNumbers => GET: /invoices/booked
 * 2. Get all invoices by number => GET invoices/booked/{num}
 * 3. Store Lines info ()
 */

// InitInvoiceOutput starts the whole thing :-)
func InitInvoiceOutput() {
	if err := environment(); err != nil {
		logger.Fatalf("Error with env: %s", err)
	}
	// Get economics invoices data requests => struct
	invoices, err := getEconomicsBookedInvoices()
	if err != nil {
		log.Fatalf("Couldnt get the booked invoices: %s", err)
	}

	// For each invoices (BookedINvoices), fetch the corresponding specific invoice line /invoices/booked/{number}
	invoice, err := getEconomicsBookedInvoice(invoices)
	if err != nil {
		log.Fatalf("Couldnt get the booked invoice: %s", err)
	}
	_ = invoice

}

func getEconomicsBookedInvoices() ([]*BookedInvoice, error) {
	invoices := BookedInvoices{}
	url := ecoURL + "/invoices/booked"
	res := createReq(url)
	err := json.NewDecoder(res.Body).Decode(&invoices)
	if err != nil {
		return nil, err
	}
	return invoices.Collection, nil
}

func getEconomicsBookedInvoice(invoices []*BookedInvoice) (*BookedInvoice, error) {
	invoice := BookedInvoice{}
	for idx, val := range invoices {
		url := ecoURL + "/invoices/booked/" + strconv.Itoa(val.BookedInvoiceNumber)
		res := createReq(url)
		err := json.NewDecoder(res.Body).Decode(&invoice)
		if err != nil {
			return nil, err
		}
		lines, err := getLineCreditValues(&invoice)
		if err != nil {
			log.Panicf("Couldnt get the line values: %s", err)
		}

		// Set PDFData values based on invoice values
		if err := WritePDF(lines, &invoice); err != nil {
			log.Fatalf("Couldn't write PDF :-(: %s", err)
		}
		fmt.Printf("Got #%s invoice with customer: %s\n", strconv.Itoa(idx), val.Recipient.Name)
	}
	return &invoice, nil
}

func getLineCreditValues(invoice *BookedInvoice) ([]*Lines, error) {
	numOfLines := len(invoice.Lines)
	lines := []*Lines{}
	if numOfLines < creditLineNumber {
		return nil, errors.New("There's no booking of credit amount on this invoice")
	}
	for _, val := range invoice.Lines {
		if val.LineNumber == creditLineNumber {
			// fmt.Printf("This line: %v has credits: %v\n", val.Description, val.CreditQuantity)
			lines = append(lines, val)
		}
	}
	fmt.Printf("%+v\n", lines)
	return lines, nil
}

func createReq(url string) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error with request setup: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-AppSecretToken", os.Getenv("ECONOMIC_SECRET_TOKEN"))
	req.Header.Add("X-AgreementGrantToken", os.Getenv("ECONOMIC_PUBLIC_TOKEN"))
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error with client HTTP: %s", err)
	}
	// defer res.Body.Close()
	return res
}

func environment() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	return nil
}

func printStructAsJSONText(i interface{}) {
	format, _ := json.Marshal(i)
	fmt.Println(string(format))
}