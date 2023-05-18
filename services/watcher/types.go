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
}

type AddressData struct {
	Data    []Tx    `json:"data"`
	Account Account `json:"account"`
}

type PriceData struct {
	Data struct {
		PriceUSD float64 `json:"price_usd"`
	} `json:"data"`
}
