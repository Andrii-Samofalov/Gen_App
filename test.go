package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

type btcTrade struct {
	Btc_uah struct {
		Sell           string `json:"sell"`
		Currency_trade string `json:"currency_trade"`
		Buy_usd        string `json:"buy_usd"`
		Buy            string `json:"buy"`
		Last           string `json:"last"`
		Updated        string `json:"updated"`
		Vol            string `json:"vol"`
		Sell_usd       string `json:"sell_usd"`
		Last_usd       string `json:"last_usd"`
		Currency_base  string `json:"currency_base"`
		Vol_cur        string `json:"vol_cur"`
		High           string `json:"high"`
		Low            string `json:"low"`
		Vol_cur_usd    string `json:"vol_cur_usd"`
		Avg            string `json:"avg"`
		Usd_rate       string `json:"usd_rate"`
	} `json:"btc_uah"`
}

func getContent(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func getBitcoinPrice() string {
	apiUrl := "https://btc-trade.com.ua/api/ticker/btc_uah"

	content, err := getContent(apiUrl)
	if err != nil {
		fmt.Println("Could not get bitcoin price from btc-trade")
		return ""
	}

	var bitcoinUAH btcTrade
	json.Unmarshal(content, &bitcoinUAH)

	return bitcoinUAH.Btc_uah.Sell
}

func rateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	price := getBitcoinPrice()

	if price == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(price))
}

func subscribeHandler(formData string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		file, err := os.OpenFile("emails.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			fmt.Println("Could not open file 'emails.txt'")
		}
		defer file.Close()

		var emails []string

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			emails = append(emails, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Could not read from file 'emails.txt'")
		}

		for i := range emails {
			if emails[i] == formData {
				w.WriteHeader(http.StatusConflict)
				return
			}
		}
		_, err2 := file.WriteString(formData + "\n")

		if err2 != nil {
			fmt.Println("Could not add email to 'emails.txt'")
		}

		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fn)
}

func sendEmailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	price := getBitcoinPrice()

	file, err := os.OpenFile("emails.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		fmt.Println("Could not open file 'emails.txt'")
	}
	defer file.Close()

	var emails []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		emails = append(emails, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Could not read from file 'emails.txt'")
	}

	var unsentEmails []string
	for i := range emails {
		from := "Account@test.com"
		password := "Account password"

		smtpHost := "smtp.gmail.com"
		smtpPort := "587"

		message := fmt.Sprintf("Current bitcoin price is %s UAH", price)

		auth := smtp.PlainAuth("", from, password, smtpHost)

		err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, strings.Split(emails[i], ""), []byte(message))
		if err != nil {
			fmt.Println(err)
			unsentEmails = append(unsentEmails, emails[i])
		}
	}
	w.WriteHeader(http.StatusOK)

	var unsent []byte
	for i := range unsentEmails {
		unsent = append(unsent, []byte(unsentEmails[i])...)
		unsent = append(unsent, "\n"...)
	}
	w.Write(unsent)
}

func main() {

	mux := http.NewServeMux()

	basicURL := "gses2.app/api"

	rh := http.HandlerFunc(rateHandler)
	mux.Handle("/rate", rh)

	var newEmail string
	sh := subscribeHandler(newEmail)
	mux.Handle("/subscribe", sh)

	seh := http.HandlerFunc(sendEmailsHandler)
	mux.Handle("/sendEmails", seh)

	http.ListenAndServe(basicURL+":3000", mux)
}
