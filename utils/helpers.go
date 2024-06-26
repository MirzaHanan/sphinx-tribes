package utils

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strconv"
	"time"

	decodepay "github.com/nbd-wtf/ln-decodepay"
)

func GetRandomToken(length int) string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println("Random token erorr ==", err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes)[:length]
}

func ConvertStringToUint(number string) (uint, error) {
	numberParse, err := strconv.ParseUint(number, 10, 32)

	if err != nil {
		fmt.Println("could not parse string to uint")
		return 0, err
	}

	return uint(numberParse), nil
}

func ConvertStringToInt(number string) (int, error) {
	numberParse, err := strconv.ParseInt(number, 10, 32)

	if err != nil {
		fmt.Println("could not parse string to uint")
		return 0, err
	}

	return int(numberParse), nil
}

func GetInvoiceAmount(paymentRequest string) uint {
	decodedInvoice, err := decodepay.Decodepay(paymentRequest)

	if err != nil {
		fmt.Println("Could not Decode Invoice", err)
		return 0
	}
	amountInt := decodedInvoice.MSatoshi / 1000
	amount := uint(amountInt)

	return amount
}

func GetInvoiceExpired(paymentRequest string) bool {
	decodedInvoice, err := decodepay.Decodepay(paymentRequest)
	if err != nil {
		fmt.Println("Could not Decode Invoice", err)
		return false
	}

	timeInUnix := time.Now().Unix()
	expiryDate := decodedInvoice.CreatedAt + decodedInvoice.Expiry

	if timeInUnix > int64(expiryDate) {
		return true
	} else {
		return false
	}
}

func GetDateDaysDifference(createdDate int64, paidDate *time.Time) int64 {
	firstDate := time.Unix(createdDate, 0)
	difference := paidDate.Sub(*&firstDate)
	days := int64(difference.Hours() / 24)
	return days
}
