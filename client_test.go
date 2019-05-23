package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const SecretKey = "12345"

type Row struct {
	ID       int    `xml:"id"`
	Age      int    `xml:"age"`
	Name     string `xml:"first_name"`
	LastName string `xml:"last_name"`
	Gender   string `xml:"gender"`
	About    string `xml:"about"`
}
type Rows struct {
	Version string `xml:"version,attr"`
	List    []Row  `xml:"row"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/*
SearchServer - своего рода внешняя система.
Непосредственно занимается поиском данных в файле `dataset.xml`. В продакшене бы запускалась в виде отдельного веб-сервиса.
* Параметр `query` ищет по полям `Name` и `About`
* Параметр `order_field` работает по полям `Id`, `Age`, `Name`, если пустой - то возвращаем по `Name`, если что-то другое - SearchServer ругается ошибкой.
 `Name` - это first_name + last_name из xml.
* Если `query` пустой, то делаем только сортировку, т.е. возвращаем все записи
*/
func SearchServer(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	AToken := r.Header.Get("AccessToken")
	if AToken != SecretKey {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"Error": "StatusUnauthorized"}`)
		return
	}

	query := r.FormValue("query")
	orderField := r.FormValue("order_field")
	limit, _ := strconv.Atoi(r.FormValue("limit"))
	offset, _ := strconv.Atoi(r.FormValue("offset"))

	if orderField == "" {
		orderField = "Name"
	}
	if orderField != "Name" && orderField != "Id" && orderField != "Age" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"Error": "ErrorBadOrderField"}`)
		return
	}

	if query == "need500" {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"Error": "StatusInternalServerError"}`)
		return
	}
	if query == "AnotherBadRequest" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"Error": "AnotherBadRequest"}`)
		return
	}
	if query == "UnpackJsonErrorBadRequest" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"}`)
		return
	}
	if query == "UnpackJsonErrorBadJson" {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"}`)
		return
	}

	var searchResult []Row

	xmlData, e := ioutil.ReadFile("dataset.xml")
	if e != nil {
		panic(e)
	}

	allData := new(Rows)
	err := xml.Unmarshal(xmlData, &allData)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	if query != "" {
		for _, u := range allData.List {
			if strings.Contains(u.Name, query) || strings.Contains(u.LastName, query) || strings.Contains(u.About, query) {
				searchResult = append(searchResult, u)
			}
		}
	} else {
		searchResult = allData.List
	}

	switch orderField {
	case "Name":
		sort.Slice(searchResult, func(i, j int) bool {
			return searchResult[i].Name < searchResult[j].Name
		})
	case "Age":
		sort.Slice(searchResult, func(i, j int) bool {
			return searchResult[i].Age < searchResult[j].Age
		})
	case "Id":
		sort.Slice(searchResult, func(i, j int) bool {
			return searchResult[i].ID < searchResult[j].ID
		})
	}

	if query == "UnknownError" {
		w.WriteHeader(http.StatusOK)
		//тут пыталась увеличить покрытие выходя за границы, но тогда тест не проходит
		//searchResult = searchResult[offset : offset+limit]
		return
	}

	searchResult = searchResult[min(len(searchResult), offset):min(len(searchResult), offset+limit)]
	var usersResult []User
	for _, u := range searchResult {
		currUser := User{
			Id:     u.ID,
			Name:   u.Name + " " + u.LastName,
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		}
		usersResult = append(usersResult, currUser)
	}

	result, err := json.Marshal(usersResult)

	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(result)

	return

}

type TestCase struct {
	SRequest SearchRequest
	Result   *SearchResponse
	IsError  bool
}

func TestAutorisation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	c := &SearchClient{
		AccessToken: "bad12345",
		URL:         ts.URL,
	}
	result, err := c.FindUsers(SearchRequest{
		Limit: 1,
	})

	if err == nil {
		t.Errorf("TestAutorisation: expected Autorisation error")
	}
	if result != nil {
		t.Errorf("TestAutorisation doesn't expect result")
	}

}

func Get25Users() *SearchResponse {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	c := &SearchClient{
		AccessToken: SecretKey,
		URL:         ts.URL,
	}
	result, _ := c.FindUsers(SearchRequest{
		Limit: 25,
	})

	return result

}

func TestFindUsers(t *testing.T) {
	findedUsers1 := make([]User, 1)
	findedUsers1[0] = User{
		Id:     1,
		Name:   "Hilda Mayer",
		Age:    21,
		About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
		Gender: "female",
	}

	findedUsers2 := make([]User, 2)
	findedUsers2[0] = User{
		Id:     1,
		Name:   "Hilda Mayer",
		Age:    21,
		About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
		Gender: "female",
	}
	findedUsers2[1] = User{
		Id:     6,
		Name:   "Jennings Mays",
		Age:    39,
		About:  "Veniam consectetur non non aliquip exercitation quis qui. Aliquip duis ut ad commodo consequat ipsum cupidatat id anim voluptate deserunt enim laboris. Sunt nostrud voluptate do est tempor esse anim pariatur. Ea do amet Lorem in mollit ipsum irure Lorem exercitation. Exercitation deserunt adipisicing nulla aute ex amet sint tempor incididunt magna. Quis et consectetur dolor nulla reprehenderit culpa laboris voluptate ut mollit. Qui ipsum nisi ullamco sit exercitation nisi magna fugiat anim consectetur officia.\n",
		Gender: "male",
	}

	cases := []TestCase{
		TestCase{
			SRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "May",
				OrderField: "Age1",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Limit: -1,
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Limit: 30,
			},
			Result:  Get25Users(),
			IsError: false,
		},
		TestCase{
			SRequest: SearchRequest{
				Offset: -1,
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Query: "need500",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Query: "AnotherBadRequest",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Query: "UnpackJsonErrorBadRequest",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Query: "UnpackJsonErrorBadJson",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Query: "UnknownError",
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			SRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "May",
				OrderField: "Age",
			},
			Result: &SearchResponse{
				Users:    findedUsers1,
				NextPage: true,
			},
			IsError: false,
		},
		TestCase{
			SRequest: SearchRequest{
				Limit:  0,
				Offset: 0,
			},
			Result: &SearchResponse{
				Users:    []User{},
				NextPage: true,
			},
			IsError: false,
		},
		TestCase{
			SRequest: SearchRequest{
				Limit:      2,
				Offset:     0,
				Query:      "May",
				OrderField: "Age",
			},
			Result: &SearchResponse{
				Users:    findedUsers2,
				NextPage: false,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken: SecretKey,
			URL:         ts.URL,
		}

		result, err := c.FindUsers(item.SRequest)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, \nexpected -> \n%#v, \n got-> \n%#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}
