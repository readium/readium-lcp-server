package main

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/abbot/go-http-auth"
)

func main() {

	var salt []byte
	var magic []byte
	if len(os.Args) < 2 {
		panic("need a password")
	}
	if len(os.Args) > 2 {
		salt = []byte(os.Args[2])
	} else {
		r := rand.New(rand.NewSource(int64(time.Now().Unix())))
		salt = []byte(strconv.Itoa(r.Int()))
	}
	if len(os.Args) > 3 {
		magic = []byte("$" + string(os.Args[3]) + "$")
	} else {
		magic = []byte("$" + "$")
	}

	log.Println(string(auth.MD5Crypt([]byte(os.Args[1]), salt, magic)))

}
