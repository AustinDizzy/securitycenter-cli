package auth

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/fatih/color"

	"github.com/boltdb/bolt"
	"github.com/howeyc/gopass"
	"github.com/urfave/cli"
)

const (
	//BucketName is the name of the bucket created in the bolt database
	//to store authentication information. The BucketName is prepended to the
	//uri of the SecurityCenter instance.
	BucketName = "AuthBucket"
	//DB is the name of the bolt database file
	DB = "auth.db"
	//ETC is the Estimated Time to Complete session (e.g. how long until session self-destructs)
	ETC = 60 * time.Minute
)

//Get returns the current token and sesion information for the current
//session if present. First, it will look at the flags passed to the binary at
//runtime (e.g. --token and --session). If those don't exist, it will look in
//the local bolt database file (auth.db).
func Get(c *cli.Context) (map[string]string, error) {
	var (
		i      int64
		db     *bolt.DB
		err    error
		data   = map[string]string{}
		bucket = []byte(fmt.Sprintf("%s|%s", BucketName, c.GlobalString("host")))
	)

	if c.GlobalIsSet("token") && c.GlobalIsSet("session") {
		data["token"] = c.GlobalString("token")
		data["session"] = c.GlobalString("session")
		return data, nil
	}

	db, err = initDB(c)
	utils.LogErr(c, err)

	if err != nil {
		return data, err
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).ForEach(func(k, v []byte) error {
			key, val := bytes.NewBuffer(k).String(), bytes.NewBuffer(v).String()
			data[key] = val
			return nil
		})
	})
	utils.LogErr(c, err, data)

	if t, ok := data["__timestamp"]; ok {
		i, err = strconv.ParseInt(t, 10, 64)
		if time.Since(time.Unix(i, 0)) > ETC {
			err = Delete(c)
			println("Your session has expired.")
			data = nil
		}
	}

	return data, err
}

//Set sets the session information based on the supplied `keys` map.
//These values are transalted as a key-value pair and are saved accordingly to
//the local auth.db bolt database file. On save, it sets the current `__timestamp`
//field to auto invalidate the session after a specified time.
func Set(c *cli.Context, keys map[string]string) {
	var (
		k, v    string
		db, err = bolt.Open(DB, 0600, nil)
		bucket  = fmt.Sprintf("%s|%s", BucketName, c.GlobalString("host"))
	)
	utils.LogErr(c, err)
	defer db.Close()

	keys["__timestamp"] = fmt.Sprint(time.Now().Unix())

	for k, v = range keys {
		err = db.Update(func(tx *bolt.Tx) error {
			b, _ := tx.CreateBucketIfNotExists([]byte(bucket))
			return b.Put([]byte(k), []byte(v))
		})
		utils.LogErr(c, err, k, v)
	}
}

//Do begins the interactive login process
func Do(c *cli.Context) {
	var (
		reader   = bufio.NewReader(os.Stdin)
		username string
		password []byte
		data     = map[string]string{}
		err      error
		jsonStr  []byte
		msg      = fmt.Sprintf("You are logging into %s", color.CyanString("%s", c.GlobalString("host"))) +
			fmt.Sprintf("\nYour session will self-destruct in %s", color.RedString("%s", ETC))
		sessionRgx = regexp.MustCompile(`(?:TNS_SESSIONID=)(.*?)(?:;)`)
	)

	fmt.Fprintf(color.Output, "%s\n\nUsername: ", msg)
	username, _ = reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Printf("Password: ")
	if c.GlobalBool("debug") {
		password, err = gopass.GetPasswdMasked()
	} else {
		password, err = gopass.GetPasswd()
	}
	utils.LogErr(c, err)

	res, err := api.NewRequest("GET", "system").Do(c)
	utils.LogErr(c, err)

	if mtch := sessionRgx.FindStringSubmatch(res.HTTPRes.Header.Get("Set-Cookie")); len(mtch) > 1 {
		data["session"] = mtch[1]
	}

	res, err = api.NewRequest("POST", "token", map[string]interface{}{
		"password": string(password[:]),
		"username": username,
	}).WithAuth(data).Do(c)

	jsonStr, err = res.Data.MarshalJSON()
	utils.LogErr(c, err, string(jsonStr[:]))

	t := res.Data.Get("response").Get("token").Interface()
	if len(fmt.Sprint(t)) > 0 {
		data["token"] = fmt.Sprint(t)
		Set(c, data)
	}
	fmt.Printf("session: %s, token: %s\n", data["session"], data["token"])
}

//Delete will purge the local bolt database. Note this does not invalidate the
//session from SecurityCenter incase the currently used token is being used
//elsewhere. An option for also invalidating the session may be present in
//future iteration(s).
func Delete(c *cli.Context) error {
	var (
		db, err = bolt.Open(DB, 0600, nil)
		bucket  = fmt.Sprintf("%s|%s", BucketName, c.GlobalString("host"))
	)
	defer db.Close()
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(bucket))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
}

//Test tests the current authentication information by making a simple
//request to the configured SecurityCenter instance. If the user is successfully
//authenticated, they will be greeted with their username and fullname while the
//function returns `true`. If not, they are shown an error and the function
//returns false.
func Test(c *cli.Context) (ok bool) {
	var (
		fields = map[string]interface{}{
			"fields": "firstname,lastname,username",
		}
		keys, err = Get(c)
		res       *api.Result
	)

	utils.LogErr(c, err)

	if err != nil {
		utils.LogErr(c, err, res)
		return false
	}

	res, err = api.NewRequest("GET", "currentUser", fields).WithAuth(keys).Do(c)
	ok = err == nil && res.Status == 200 && res.Data != nil
	utils.LogErr(c, err, res, keys)

	if ok {
		data := res.Data.Get("response")
		fmt.Printf("Hello %s %s (%s).\n", data.Get("firstname").MustString(), data.Get("lastname").MustString(), data.Get("username").MustString())
	} else {
		fmt.Printf("No valid authentication. Please run `%s auth` to start your session.", c.App.Name)
	}

	return ok
}

func initDB(c *cli.Context) (*bolt.DB, error) {
	var (
		db     *bolt.DB
		err    error
		bucket = []byte(fmt.Sprint(BucketName, "|", c.GlobalString("host")))
	)

	db, err = bolt.Open(DB, 0666, nil)

	if err != nil {
		return db, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		var (
			b         *bolt.Bucket
			bucketErr = errors.New("Error creating bucket.")
		)
		if tx.Writable() {
			b, _ = tx.CreateBucketIfNotExists(bucket)
			if b == nil {
				return bucketErr
			}
		} else {
			return bolt.ErrTxNotWritable
		}
		if b != nil {
			return nil
		}
		return bucketErr
	})

	return db, err
}
