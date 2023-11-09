package api

type Item struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type Server struct {
	shoppingBag []Item
}
