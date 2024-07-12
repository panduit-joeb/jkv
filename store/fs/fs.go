package fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/panduit-joeb/jkv"
)

type Options struct {
	Addr, Password string
	DB             int
}

type Client struct {
	DBDir  string
	IsOpen bool
}

var _ jkv.Client = (*Client)(nil)

const DEFAULT_DB = "jkv_db"

func (j *Client) ScalarDir() string { return j.DBDir + "/scalars/" }
func (j *Client) HashDir() string   { return j.DBDir + "/hashes/" }
func notOpen() error                { return errors.New("DB is not open") }

func NewClient(opts *Options) (db *Client) {
	return &Client{DBDir: opts.Addr, IsOpen: false}
}

// Open a database by creating the directories required if they don't exist and mark the database open
func (j *Client) Open() error {
	j.IsOpen = false
	for _, dir := range []string{j.ScalarDir(), j.HashDir()} {
		if err := os.MkdirAll(dir, 0775); err != nil {
			return err
		}
	}
	j.IsOpen = true
	return nil
}

// Close a database, basically just mark it closed
func (j *Client) Close() { j.IsOpen = false }

// FLUSHDB a database by removing the j.dbDir and everything underneath, ignore errors for now
func (j *Client) FlushDB() { os.RemoveAll(j.DBDir) }

// Return data in scalar key data, error is file is missing or inaccessible
func (c *Client) Get(ctx context.Context, key string) *jkv.StringCmd {
	if c.IsOpen {
		data, err := os.ReadFile(c.ScalarDir() + key)
		return jkv.NewStringCmd(string(data), err)
	}
	return jkv.NewStringCmd("", notOpen())
}

// Set a scalar key to a value
func (c *Client) Set(ctx context.Context, key, value string) *jkv.StatusCmd {
	if c.IsOpen {
		return jkv.NewStatusCmd("OK", os.WriteFile(c.DBDir+"/scalars/"+key, []byte(value), 0660))
	}
	return jkv.NewStatusCmd("", notOpen())
}

// Delete a key by removing the scalar file
func (c *Client) Del(ctx context.Context, keys ...string) *jkv.IntCmd {
	if c.IsOpen {
		// todo: add a loop here
		return jkv.NewIntCmd(int64(len(keys)), os.Remove(c.ScalarDir()+keys[0]))
	}
	return jkv.NewIntCmd(0, notOpen())
}

// KEYS returns the scalar and hash keys
func (c *Client) Keys(ctx context.Context, pattern string) *jkv.StringSliceCmd {
	var files []string
	for _, dir := range []string{c.ScalarDir(), c.HashDir()} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return jkv.NewStringSliceCmd([]string{}, err)
		}
		for _, file := range entries {
			files = append(files, file.Name())
		}
	}
	return jkv.NewStringSliceCmd(files, nil)
}

// Return true if scalar key file exists, false otherwise
func (c *Client) Exists(ctx context.Context, keys ...string) *jkv.IntCmd {
	if c.IsOpen {
		// todo: add a loop here
		if _, err := os.Stat(c.ScalarDir() + keys[0]); err != nil {
			return jkv.NewIntCmd(0, err)
		}
		// return jkv.NewIntCmd(int64(len(keys)), nil)
		return jkv.NewIntCmd(1, nil)
	}
	return jkv.NewIntCmd(0, nil)
}

// Return data in hashed key data, error is file is missing or inaccessible
func (c *Client) HGet(ctx context.Context, hash, key string) *jkv.StringCmd {
	if c.IsOpen {
		data, err := os.ReadFile(c.HashDir() + hash + "/" + key)
		if err != nil {
			return jkv.NewStringCmd("", err)
		}
		return jkv.NewStringCmd(string(data), nil)
	}
	return jkv.NewStringCmd("", notOpen())
}

// Create a hash directory and store the data in a key file
// todo: reject a hash if a scalar key exists
func (c *Client) HSet(ctx context.Context, hash, key, value string) *jkv.IntCmd {
	if c.IsOpen {
		rec := c.Exists(ctx, hash)
		if rec.Err() != nil {
			return jkv.NewIntCmd(0, rec.Err())
		}
		if rec.Val() > 0 {
			return jkv.NewIntCmd(0, fmt.Errorf("key \"%s\" exists as a scalar, cannot be a hash", hash))
		}
		if err := os.MkdirAll(c.HashDir()+hash, 0775); err != nil {
			return jkv.NewIntCmd(0, rec.Err())
		}
		if err := os.WriteFile(c.HashDir()+hash+"/"+key, []byte(value), 0664); err != nil {
			return jkv.NewIntCmd(0, rec.Err())
		}
		jkv.NewIntCmd(1, nil)
	}
	return jkv.NewIntCmd(0, notOpen())
}

// Delete a hashed key by removing the file, if no keys exist after the operation remove the hash directory
func (c *Client) HDel(ctx context.Context, hash, key string) *jkv.IntCmd {
	var err error
	var entries []fs.DirEntry

	if c.IsOpen {
		if err = os.Remove(c.HashDir() + hash + "/" + key); err != nil {
			return jkv.NewIntCmd(0, err)
		}
		if entries, err = os.ReadDir(c.HashDir() + hash); err != nil {
			return jkv.NewIntCmd(0, err)
		}
		if len(entries) == 0 {
			err = os.RemoveAll(c.HashDir() + hash)
			if err != nil {
				return jkv.NewIntCmd(0, err)
			}
		}
		return jkv.NewIntCmd(int64(len(entries)), err)
	}
	return jkv.NewIntCmd(0, notOpen())
}

// HKEYS returns the hash keys
func (c *Client) HKeys(ctx context.Context, hash string) *jkv.StringSliceCmd {
	var err error
	if c.IsOpen {
		if _, err = os.Stat(c.HashDir() + hash); err == nil {
			entries, err := os.ReadDir(c.HashDir() + hash)
			if err != nil {
				return jkv.NewStringSliceCmd([]string{}, err)
			}
			var files []string
			for _, file := range entries {
				files = append(files, file.Name())
			}
			return jkv.NewStringSliceCmd(files, nil)
		}
		return jkv.NewStringSliceCmd([]string{}, err)
	}
	return jkv.NewStringSliceCmd([]string{}, notOpen())
}

// Return true if hashed key file exists, false otherwise
func (c *Client) HExists(ctx context.Context, hash, key string) *jkv.BoolCmd {
	if c.IsOpen {
		var err error
		if _, err = os.Stat(c.HashDir() + hash + "/" + key); err != nil {
			return jkv.NewBoolCmd(false, err)
		}
		return jkv.NewBoolCmd(true, nil)
	}
	return jkv.NewBoolCmd(false, notOpen())
}

func (c *Client) Ping(ctx context.Context) *jkv.StatusCmd {
	if c.IsOpen {
		return jkv.NewStatusCmd("PONG", nil)
	}
	return jkv.NewStatusCmd("", notOpen())
}
