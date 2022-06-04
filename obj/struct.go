package obj

import "time"

var List = []interface{}{new(User), new(Job)}

type LocalTime time.Time

// User UserInfo
type User struct {
	Id   string `json:"id"`   // id field
	Name string `json:"name"` // username
	Age  int    `json:"age"`  // user age
}

type Job struct {
	Id         string    `json:"id"` // id field
	Type       string    `json:"type"`
	Content    string    `json:"content"`
	CreateTime LocalTime `json:"create_time"`
	UpdateTime LocalTime `json:"update_time"`
}
