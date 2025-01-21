package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/stakwork/sphinx-tribes/auth"
	"github.com/stakwork/sphinx-tribes/config"
	"github.com/stakwork/sphinx-tribes/db"
	"github.com/stakwork/sphinx-tribes/logger"
	"github.com/stakwork/sphinx-tribes/utils"
)

type tribeHandler struct {
	db                      db.Database
	verifyTribeUUID         func(uuid string, checkTimestamp bool) (string, error)
	tribeUniqueNameFromName func(name string) (string, error)
}

func NewTribeHandler(db db.Database) *tribeHandler {
	return &tribeHandler{
		db:                      db,
		verifyTribeUUID:         auth.VerifyTribeUUID,
		tribeUniqueNameFromName: TribeUniqueNameFromName,
	}
}

func (th *tribeHandler) GetAllTribes(w http.ResponseWriter, r *http.Request) {
	tribes := th.db.GetAllTribes()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribes)
}

func (th *tribeHandler) GetTotalribes(w http.ResponseWriter, r *http.Request) {
	tribesTotal := th.db.GetTribesTotal()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribesTotal)
}

func (th *tribeHandler) GetListedTribes(w http.ResponseWriter, r *http.Request) {
	tribes := th.db.GetListedTribes(r)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribes)
}

func (th *tribeHandler) GetTribesByOwner(w http.ResponseWriter, r *http.Request) {
	all := r.URL.Query().Get("all")
	tribes := []db.Tribe{}
	pubkey := chi.URLParam(r, "pubkey")
	if all == "true" {
		tribes = th.db.GetAllTribesByOwner(pubkey)
	} else {
		tribes = th.db.GetTribesByOwner(pubkey)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribes)
}

func (th *tribeHandler) GetTribesByAppUrl(w http.ResponseWriter, r *http.Request) {
	tribes := []db.Tribe{}
	app_url := chi.URLParam(r, "app_url")
	tribes = th.db.GetTribesByAppUrl(app_url)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribes)
}

func GetTribesByAppUrls(w http.ResponseWriter, r *http.Request) {
	app_urls := chi.URLParam(r, "app_urls")
	app_url_list := strings.Split(app_urls, ",")
	m := make(map[string][]db.Tribe)
	for _, app_url := range app_url_list {
		tribes := db.DB.GetTribesByAppUrl(app_url)
		m[app_url] = tribes
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

func PutTribeStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	tribe := db.Tribe{}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	err = json.Unmarshal(body, &tribe)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	if tribe.UUID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := auth.VerifyTribeUUID(tribe.UUID, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	now := time.Now()
	tribe.Updated = &now
	db.DB.UpdateTribe(tribe.UUID, map[string]interface{}{
		"member_count": tribe.MemberCount,
		"updated":      &now,
		"bots":         tribe.Bots,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func (th *tribeHandler) DeleteTribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	uuid := chi.URLParam(r, "uuid")

	if uuid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := th.verifyTribeUUID(uuid, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	th.db.UpdateTribe(uuid, map[string]interface{}{
		"deleted": true,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func (th *tribeHandler) GetTribe(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "uuid")
	tribe := th.db.GetTribe(uuid)

	var theTribe map[string]interface{}
	j, _ := json.Marshal(tribe)
	json.Unmarshal(j, &theTribe)

	theTribe["channels"] = th.db.GetChannelsByTribe(uuid)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(theTribe)
}

func (th *tribeHandler) GetFirstTribeByFeed(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	tribe := th.db.GetFirstTribeByFeedURL(url)

	if tribe.UUID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var theTribe map[string]interface{}
	j, _ := json.Marshal(tribe)
	json.Unmarshal(j, &theTribe)

	theTribe["channels"] = th.db.GetChannelsByTribe(tribe.UUID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(theTribe)
}

func (th *tribeHandler) GetTribeByUniqueName(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "un")
	tribe := th.db.GetTribeByUniqueName(uuid)

	var theTribe map[string]interface{}
	j, _ := json.Marshal(tribe)
	json.Unmarshal(j, &theTribe)

	theTribe["channels"] = th.db.GetChannelsByTribe(tribe.UUID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(theTribe)
}

func (th *tribeHandler) CreateOrEditTribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	tribe := db.Tribe{}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	err = json.Unmarshal(body, &tribe)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	if tribe.UUID == "" {
		logger.Log.Info("createOrEditTribe no uuid")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	now := time.Now() //.Format(time.RFC3339)

	extractedPubkey, err := th.verifyTribeUUID(tribe.UUID, false)
	if err != nil {
		logger.Log.Error("extract UUID error: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if pubKeyFromAuth == "" {
		tribe.Created = &now
	} else { // IF PUBKEY IN CONTEXT, MUST AUTH!
		if pubKeyFromAuth != extractedPubkey {
			logger.Log.Info("createOrEditTribe pubkeys dont match")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	existing := th.db.GetTribe(tribe.UUID)
	if existing.UUID == "" { // if doesn't exist already, create unique name
		tribe.UniqueName, _ = th.tribeUniqueNameFromName(tribe.Name)
	} else { // already exists! make sure it's owned
		if existing.OwnerPubKey != extractedPubkey {
			logger.Log.Info("createOrEditTribe tribe.ownerPubKey not match")
			logger.Log.Info("existing owner: %s", existing.OwnerPubKey)
			logger.Log.Info("extracted pubkey: %s", extractedPubkey)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	tribe.OwnerPubKey = extractedPubkey
	tribe.Updated = &now
	tribe.LastActive = now.Unix()

	_, err = th.db.CreateOrEditTribe(tribe)
	if err != nil {
		logger.Log.Error("=> ERR createOrEditTribe: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tribe)
}

func PutTribeActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := auth.VerifyTribeUUID(uuid, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	now := time.Now().Unix()
	db.DB.UpdateTribe(uuid, map[string]interface{}{
		"last_active": now,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func (th *tribeHandler) SetTribePreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)

	uuid := chi.URLParam(r, "uuid")
	if uuid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := th.verifyTribeUUID(uuid, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	preview := r.URL.Query().Get("preview")
	th.db.UpdateTribe(uuid, map[string]interface{}{
		"preview": preview,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func CreateLeaderBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	uuid := chi.URLParam(r, "tribe_uuid")

	leaderBoard := []db.LeaderBoard{}

	if uuid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := auth.VerifyTribeUUID(uuid, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	err = json.Unmarshal(body, &leaderBoard)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	_, err = db.DB.CreateLeaderBoard(uuid, leaderBoard)

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func GetLeaderBoard(w http.ResponseWriter, r *http.Request) {
	uuid := chi.URLParam(r, "tribe_uuid")
	alias := r.URL.Query().Get("alias")

	if alias == "" {
		leaderBoards := db.DB.GetLeaderBoard(uuid)

		var board = []db.LeaderBoard{}
		for _, leaderboard := range leaderBoards {
			leaderboard.TribeUuid = ""
			board = append(board, leaderboard)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(board)
	} else {
		leaderBoardFromDb := db.DB.GetLeaderBoardByUuidAndAlias(uuid, alias)

		if leaderBoardFromDb.Alias != alias {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(leaderBoardFromDb)
	}
}

func UpdateLeaderBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pubKeyFromAuth, _ := ctx.Value(auth.ContextKey).(string)
	uuid := chi.URLParam(r, "tribe_uuid")

	if uuid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	extractedPubkey, err := auth.VerifyTribeUUID(uuid, false)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//from token must match
	if pubKeyFromAuth != extractedPubkey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	leaderBoard := db.LeaderBoard{}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	err = json.Unmarshal(body, &leaderBoard)
	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	leaderBoardFromDb := db.DB.GetLeaderBoardByUuidAndAlias(uuid, leaderBoard.Alias)

	if leaderBoardFromDb.Alias != leaderBoard.Alias {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	leaderBoard.TribeUuid = leaderBoardFromDb.TribeUuid

	db.DB.UpdateLeaderBoard(leaderBoardFromDb.TribeUuid, leaderBoardFromDb.Alias, map[string]interface{}{
		"spent":      leaderBoard.Spent,
		"earned":     leaderBoard.Earned,
		"reputation": leaderBoard.Reputation,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(true)
}

func GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceRes, invoiceErr := db.InvoiceResponse{}, db.InvoiceError{}

	if config.IsV2Payment {
		invoiceRes, invoiceErr = GenerateV2Invoice(w, r)
	} else {
		invoiceRes, invoiceErr = GenerateV1Invoice(w, r)
	}

	if invoiceErr.Error != "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(invoiceErr)
	}

	invoice := db.InvoiceRequest{}
	body, err := io.ReadAll(r.Body)

	r.Body.Close()

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	err = json.Unmarshal(body, &invoice)

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	pub_key := invoice.User_pubkey
	owner_key := invoice.Owner_pubkey
	date, _ := utils.ConvertStringToInt(invoice.Created)
	invoiceType := invoice.Type
	routeHint := invoice.Route_hint
	amount, _ := utils.ConvertStringToUint(invoice.Amount)

	paymentRequest := invoiceRes.Response.Invoice
	now := time.Now()

	newInvoice := db.NewInvoiceList{
		PaymentRequest: paymentRequest,
		Type:           db.InvoiceType(invoiceType),
		OwnerPubkey:    owner_key,
		Created:        &now,
		Updated:        &now,
		Status:         false,
	}

	newInvoiceData := db.UserInvoiceData{
		PaymentRequest: paymentRequest,
		Created:        date,
		Amount:         amount,
		UserPubkey:     pub_key,
		RouteHint:      routeHint,
	}

	db.DB.ProcessAddInvoice(newInvoice, newInvoiceData)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoiceRes)
}

func GenerateV1Invoice(w http.ResponseWriter, r *http.Request) (db.InvoiceResponse, db.InvoiceError) {
	invoice := db.InvoiceRequest{}
	body, err := io.ReadAll(r.Body)

	r.Body.Close()

	if err != nil {
		logger.Log.Error("%v", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	err = json.Unmarshal(body, &invoice)

	if err != nil {
		logger.Log.Error("%v", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	memo := invoice.Memo
	amount, _ := utils.ConvertStringToUint(invoice.Amount)

	url := fmt.Sprintf("%s/invoices", config.RelayUrl)

	bodyData := fmt.Sprintf(`{"amount": %d, "memo": "%s"}`, amount, memo)

	jsonBody := []byte(bodyData)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))

	req.Header.Set("x-user-token", config.RelayAuthKey)
	req.Header.Set("Content-Type", "application/json")
	res, _ := client.Do(req)

	if err != nil {
		log.Printf("Request Failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)

	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	// Unmarshal result
	invoiceRes := db.InvoiceResponse{}

	err = json.Unmarshal(body, &invoiceRes)

	if err != nil {
		log.Printf("Unmarshal body failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	return invoiceRes, db.InvoiceError{Success: true}
}

func GenerateV2Invoice(w http.ResponseWriter, r *http.Request) (db.InvoiceResponse, db.InvoiceError) {
	invoice := db.InvoiceRequest{}

	var err error
	body, err := io.ReadAll(r.Body)

	r.Body.Close()

	if err != nil {
		logger.Log.Error("%v", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	err = json.Unmarshal(body, &invoice)

	if err != nil {
		logger.Log.Error("%v", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	url := fmt.Sprintf("%s/invoice", config.V2BotUrl)

	amount, _ := utils.ConvertStringToUint(invoice.Amount)

	amountMsat := amount * 1000

	bodyData := fmt.Sprintf(`{"amt_msat": %d}`, amountMsat)

	jsonBody := []byte(bodyData)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))

	req.Header.Set("x-admin-token", config.V2BotToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		log.Printf("Client Request Failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)

	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}

	// Unmarshal result
	v2InvoiceRes := db.V2CreateInvoiceResponse{}
	err = json.Unmarshal(body, &v2InvoiceRes)

	if err != nil {
		log.Printf("Json Unmarshal failed: %s", err)
		return db.InvoiceResponse{}, db.InvoiceError{Success: false, Error: err.Error()}
	}
	return db.InvoiceResponse{
		Response: db.Invoice{
			Invoice: v2InvoiceRes.Bolt11,
		},
	}, db.InvoiceError{Success: true}
}

func (th *tribeHandler) GenerateBudgetInvoice(w http.ResponseWriter, r *http.Request) {
	if config.IsV2Payment {
		th.GenerateV2BudgetInvoice(w, r)
	} else {
		th.GenerateV1BudgetInvoice(w, r)
	}
}

func (th *tribeHandler) GenerateV1BudgetInvoice(w http.ResponseWriter, r *http.Request) {
	invoice := db.BudgetInvoiceRequest{}

	var err error
	body, err := io.ReadAll(r.Body)

	r.Body.Close()

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	err = json.Unmarshal(body, &invoice)

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	if invoice.WorkspaceUuid == "" && invoice.OrgUuid != "" {
		invoice.WorkspaceUuid = invoice.OrgUuid
	}

	url := fmt.Sprintf("%s/invoices", config.RelayUrl)

	bodyData := fmt.Sprintf(`{"amount": %d, "memo": "%s"}`, invoice.Amount, "Budget Invoice")

	jsonBody := []byte(bodyData)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))

	req.Header.Set("x-user-token", config.RelayAuthKey)
	req.Header.Set("Content-Type", "application/json")
	res, _ := client.Do(req)

	if err != nil {
		log.Printf("Request Failed: %s", err)
		return
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)

	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return
	}

	// Unmarshal result
	invoiceRes := db.InvoiceResponse{}

	err = json.Unmarshal(body, &invoiceRes)

	if err != nil {
		log.Printf("Json Unmarshal failed: %s", err)
		return
	}

	now := time.Now()
	var paymentHistory = db.NewPaymentHistory{
		Amount:         invoice.Amount,
		WorkspaceUuid:  invoice.WorkspaceUuid,
		PaymentType:    invoice.PaymentType,
		SenderPubKey:   invoice.SenderPubKey,
		ReceiverPubKey: "",
		Created:        &now,
		Updated:        &now,
		Status:         false,
		BountyId:       0,
	}

	newInvoice := db.NewInvoiceList{
		PaymentRequest: invoiceRes.Response.Invoice,
		Type:           db.InvoiceType("BUDGET"),
		OwnerPubkey:    invoice.SenderPubKey,
		WorkspaceUuid:  invoice.WorkspaceUuid,
		Created:        &now,
		Updated:        &now,
		Status:         false,
	}

	th.db.ProcessBudgetInvoice(paymentHistory, newInvoice)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoiceRes)
}

func (th *tribeHandler) GenerateV2BudgetInvoice(w http.ResponseWriter, r *http.Request) {
	invoice := db.BudgetInvoiceRequest{}

	var err error
	body, err := io.ReadAll(r.Body)

	r.Body.Close()

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	err = json.Unmarshal(body, &invoice)

	if err != nil {
		logger.Log.Error("%v", err)
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	if invoice.WorkspaceUuid == "" && invoice.OrgUuid != "" {
		invoice.WorkspaceUuid = invoice.OrgUuid
	}

	url := fmt.Sprintf("%s/invoice", config.V2BotUrl)

	amountMsat := invoice.Amount * 1000

	bodyData := fmt.Sprintf(`{"amt_msat": %d}`, amountMsat)

	jsonBody := []byte(bodyData)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))

	req.Header.Set("x-admin-token", config.V2BotToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		log.Printf("Client Request Failed: %s", err)
		return
	}

	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)

	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return
	}

	// Unmarshal result
	v2InvoiceRes := db.V2CreateInvoiceResponse{}
	err = json.Unmarshal(body, &v2InvoiceRes)

	if err != nil {
		log.Printf("Json Unmarshal failed: %s", err)
		return
	}

	now := time.Now()
	var paymentHistory = db.NewPaymentHistory{
		Amount:         invoice.Amount,
		WorkspaceUuid:  invoice.WorkspaceUuid,
		PaymentType:    invoice.PaymentType,
		SenderPubKey:   invoice.SenderPubKey,
		ReceiverPubKey: "",
		Created:        &now,
		Updated:        &now,
		Status:         false,
		BountyId:       0,
	}

	newInvoice := db.NewInvoiceList{
		PaymentRequest: v2InvoiceRes.Bolt11,
		Type:           db.InvoiceType("BUDGET"),
		OwnerPubkey:    invoice.SenderPubKey,
		WorkspaceUuid:  invoice.WorkspaceUuid,
		Created:        &now,
		Updated:        &now,
		Status:         false,
	}

	th.db.ProcessBudgetInvoice(paymentHistory, newInvoice)

	invoiceRes := db.InvoiceResponse{
		Succcess: true,
		Response: db.Invoice{
			Invoice: v2InvoiceRes.Bolt11,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoiceRes)
}
