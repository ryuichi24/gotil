package todo


type TodoModel struct {
	Id string `json:"id"`
	Title string `json:"title"`
	Completed bool `json:"completed"`
	Description string `json:"description"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

