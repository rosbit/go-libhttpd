/**
 * go obj map. to follow the cgo rule: don't transfer the golang pointer to c.
 */
package main

import "sync"

var refs struct {
	sync.Mutex
	objs map[int32]interface{}
	next int32
}

func init() {
	refs.Lock()
	defer refs.Unlock()

	refs.objs = make(map[int32]interface{})
	refs.next = 1
}

func NewObjId(obj interface{}) int32 {
	refs.Lock()
	defer refs.Unlock()

	id := refs.next
	refs.next++
	if refs.next <= 0 {
		refs.next = 1
	}

	refs.objs[id] = obj
	return id
}

func GetObjById(id int32) interface{} {
	refs.Lock()
	defer refs.Unlock()

	return refs.objs[id]
}

func FreeObjId(id int32) interface{} {
	refs.Lock()
	defer refs.Unlock()

	obj := refs.objs[id]
	delete(refs.objs, id)

	return obj
}
