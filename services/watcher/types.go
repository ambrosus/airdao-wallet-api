package watcher

type Account struct {
	Balance struct {
		Wei   string  `json:"wei"`
		Ether float64 `json:"ether"`
	} `json:"balance"`
}

type Tx struct {
	BlockHash string `json:"block_hash"`
	From      string `json:"from"`
	To        string `json:"to"`
	Hash      string `json:"hash"`
	Value     struct {
		Ether  float64 `json:"ether"`
		Symbol *string `json:"symbol,omitempty"`
	} `json:"value"`

	Timestamp float64 `json:"timestamp"`
}

type ApiAddressData struct {
	Data    []Tx    `json:"data"`
	Account Account `json:"account"`
}

type ApiTxData struct {
	Data []Tx `json:"data"`
}

type PriceData struct {
	Data struct {
		PriceUSD float64 `json:"price_usd"`
	} `json:"data"`
}

type CGData struct {
	Prices [][]float64 `json:"prices"`
}
