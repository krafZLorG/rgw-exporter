package main

import (
	"context"
	"log"
	"sync"
	"time"

	rgw "github.com/ceph/go-ceph/rgw/admin"
)

var (
	collectUsersDuration   time.Duration
	collectUsersDurationMu sync.Mutex
)

var (
	users   []UserInfo
	usersMu sync.Mutex
)

func collectUsers(conn *rgw.API, showAllUsers bool) {
	debugLog("users collector: started")
	start := time.Now()

	var curUsers []UserInfo

	curUsersList, err := conn.GetUsers(context.Background())
	if err != nil {
		log.Println("users collector: unable to get users info")
		return
	}

	for _, v := range *curUsersList {
		curUser, err := conn.GetUser(context.Background(), rgw.User{ID: v})
		if err != nil {
			log.Println("users collector unable to get users info")
			return
		}
		user := UserInfo{curUser.ID, curUser.Tenant, curUser.DisplayName, *curUser.Suspended}
		if showAllUsers || (user.UserId == user.Tenant) {
			curUsers = append(curUsers, user)
		}
	}
	debugLog("users collector %v users", len(*curUsersList))
	usersMu.Lock()
	users = curUsers
	usersMu.Unlock()

	collectUsersDurationMu.Lock()
	collectUsersDuration = time.Since(start)
	collectUsersDurationMu.Unlock()
	debugLog("users collector sss finished in %s", collectUsersDuration)
}
