package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/panduit-joeb/jkv"
	"github.com/panduit-joeb/jkv/store/fs"
	"github.com/panduit-joeb/jkv/store/redis"
)

func main() {
	cmd := os.Args[0]
	if strings.Contains(os.Args[0], "/") {
		a := strings.Split(os.Args[0], "/")
		cmd = a[len(a)-1]
	}
	// fmt.Println("cmd is", cmd)

	var run_test, redis_cmd, fs_cmd, version, opt_x, prompt bool
	flag.BoolVar(&run_test, "t", false, "Run JKV tests")
	flag.BoolVar(&redis_cmd, "r", cmd == "redis-cli", "Run JKV tests using Redis")
	flag.BoolVar(&fs_cmd, "f", cmd == "jkv-cli", "Run JKV tests using FS")
	flag.BoolVar(&version, "v", false, "Print version")
	flag.BoolVar(&opt_x, "x", false, "Get value from stdin")
	flag.Parse()

	if version {
		fmt.Println(jkv.VERSION)
		os.Exit(0)
	}

	prompt = len(flag.Args()) == 0

	if run_test {
		c := fs.NewJKVClient()
		ok("open", c.Open)
		ok("set", func() error { return c.SET("this", "that") })
		c.FLUSHDB()

		r := redis.NewJKVClient()
		ok("open", r.Open)
		ok("set", func() error { return r.SET("this", "that") })
		r.FLUSHDB()
	} else if redis_cmd {
		r := redis.NewJKVClient()
		r.Open()

		if prompt {
			scanner := bufio.NewScanner(os.Stdin)

			fmt.Printf(r.DBDir + "> ")
			for scanner.Scan() {
				ProcessCmd(r, scanner.Text(), opt_x)
				fmt.Printf(r.DBDir + "> ")
			}

			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading input:", err)
			}
		} else {
			ProcessCmd(r, strings.Join(flag.Args(), " "), opt_x)
		}
	} else if fs_cmd {
		f := fs.NewJKVClient()
		f.Open()

		if prompt {
			scanner := bufio.NewScanner(os.Stdin)

			fmt.Printf(f.DBDir + "> ")
			for scanner.Scan() {
				ProcessCmd(f, scanner.Text(), opt_x)
				fmt.Printf(f.DBDir + "> ")
			}

			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading input:", err)
			}
		} else {
			ProcessCmd(f, strings.Join(flag.Args(), " "), opt_x)
		}
	}
}

func ok(label string, doit func() error) { fmt.Printf("%s returned %t\n", label, doit() == nil) }

func ProcessCmd(db interface{}, cmd string, opt_x bool) {
	var (
		value  string
		values []string
		err    error
	)
	tokens := strings.Fields(cmd)
	if len(tokens) == 0 {
		return
	}
	switch strings.ToUpper(tokens[0]) {
	case "FLUSHDB":
		if len(tokens) == 1 {
			if r, ok := db.(*redis.JKV_DB); ok {
				r.FLUSHDB()
			} else {
				db.(*fs.JKV_DB).FLUSHDB()
			}
			fmt.Println("OK")
		} else {
			fmt.Println("(error) ERR syntax error")
		}
	case "HGET":
		if len(tokens) == 3 {
			if r, ok := db.(*redis.JKV_DB); ok {
				value, err = r.HGET(tokens[1], tokens[2])
			} else {
				value, err = db.(*fs.JKV_DB).HGET(tokens[1], tokens[2])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				fmt.Printf("\"%s\"\n", value)
			}
		} else {
			fmt.Println("(nil)")
		}
	case "HSET":
		fmt.Println("add -x support")
		if len(tokens) > 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				if r.EXISTS(tokens[1]) {
					fmt.Println("(error) WRONGTYPE Operation against a key holding the wrong kind of value")
					return
				}
			} else {
				if db.(*fs.JKV_DB).EXISTS(tokens[1]) {
					fmt.Println("(error) WRONGTYPE Operation against a key holding the wrong kind of value")
					return
				}
			}
			if len(tokens) >= 4 && ((len(tokens)-2)%2 == 0) {
				hash := tokens[1]
				var n int
				for n = 0; n < (len(tokens)-1)/2; n++ {
					key := tokens[2+n*2]
					value = tokens[2+n*2+1]
					if r, ok := db.(*redis.JKV_DB); ok {
						err = r.HSET(hash, key, value)
					} else {
						err = db.(*fs.JKV_DB).HSET(hash, key, value)
					}
					if err != nil {
						fmt.Println(err.Error())
						return
					}
				}
				fmt.Printf("(integer) %d\n", n)
			} else {
				fmt.Println("(error) ERR wrong number of arguments for 'hset' command")
			}
		} else {
			fmt.Println("(nil)")
		}
	case "HDEL":
		if len(tokens) == 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				err = r.HDEL(tokens[1], tokens[2])
			} else {
				err = db.(*fs.JKV_DB).HDEL(tokens[1], tokens[2])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				fmt.Printf("\"%s\"\n", value)
			}
		} else {
			fmt.Println("(nil)")
		}
	case "HKEYS":
		if len(tokens) == 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				values, err = r.HKEYS(tokens[1])
			} else {
				values, err = db.(*fs.JKV_DB).HKEYS(tokens[1])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				for i, v := range values {
					fmt.Printf("%d) \"%s\"\n", i+1, v)
				}
			}
		} else {
			fmt.Println("(error) ERR wrong number of arguments for 'hkeys' command")
		}
	case "HEXISTS":
		if len(tokens) == 3 {
			var exists bool
			if r, ok := db.(*redis.JKV_DB); ok {
				exists = r.HEXISTS(tokens[1], tokens[2])
			} else {
				exists = db.(*fs.JKV_DB).HEXISTS(tokens[1], tokens[2])
			}
			if exists {
				fmt.Println("(integer) 1")
			} else {
				fmt.Println("(integer) 0")
			}
		} else {
			fmt.Println("(error) ERR wrong number of arguments for 'exists' command")
		}
	case "GET":
		if len(tokens) == 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				value, err = r.GET(tokens[1])
			} else {
				value, err = db.(*fs.JKV_DB).GET(tokens[1])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				fmt.Printf("\"%s\"\n", value)
			}
		} else {
			fmt.Println("(nil)")
		}
	case "SET":
		if opt_x {
			if len(tokens) == 2 {
				var buf = make([]byte, 1024*1024)
				var n = 0
				n, err = os.Stdin.Read(buf)
				if n == 0 {
					if err != io.EOF {
						panic(err.Error())
					}
					return
				}
				if r, ok := db.(*redis.JKV_DB); ok {
					err = r.SET(tokens[1], string(buf[:n-1]))
				} else {
					err = db.(*fs.JKV_DB).SET(tokens[1], string(buf[:n-1]))
				}
				if err != nil {
					fmt.Println("(nil)")
				} else {
					fmt.Println("OK")
				}
			} else {
				fmt.Println("(error) ERR wrong number of arguments for 'set' command")
			}
		} else {
			if len(tokens) == 3 {
				if r, ok := db.(*redis.JKV_DB); ok {
					err = r.SET(tokens[1], tokens[2])
				} else {
					err = db.(*fs.JKV_DB).SET(tokens[1], tokens[2])
				}
				if err != nil {
					fmt.Println("(nil)")
				} else {
					fmt.Println("OK")
				}
			} else {
				fmt.Println("(error) ERR wrong number of arguments for 'set' command")
			}
		}
	case "DEL":
		if len(tokens) == 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				err = r.DEL(tokens[1])
			} else {
				err = db.(*fs.JKV_DB).DEL(tokens[1])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				fmt.Printf("\"%s\"\n", value)
			}
		} else {
			fmt.Println("(nil)")
		}
	case "KEYS":
		if len(tokens) == 2 {
			if r, ok := db.(*redis.JKV_DB); ok {
				values, err = r.KEYS(tokens[1])
			} else {
				values, err = db.(*fs.JKV_DB).KEYS(tokens[1])
			}
			if err != nil {
				fmt.Println("(nil)")
			} else {
				for i, v := range values {
					fmt.Printf("%d) \"%s\"\n", i+1, v)
				}
			}
		} else {
			fmt.Println("(error) ERR wrong number of arguments for 'keys' command")
		}
	case "EXISTS":
		if len(tokens) == 2 {
			var exists bool
			if r, ok := db.(*redis.JKV_DB); ok {
				exists = r.EXISTS(tokens[1])
			} else {
				exists = db.(*fs.JKV_DB).EXISTS(tokens[1])
			}
			if exists {
				fmt.Println("(integer) 1")
			} else {
				fmt.Println("(integer) 0")
			}
		} else {
			fmt.Println("(error) ERR wrong number of arguments for 'exists' command")
		}
	default:
		fmt.Printf("(error) ERR unknown command '%s', with args beginning with:\n", tokens[0])
	}
}
