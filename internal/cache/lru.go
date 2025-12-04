package cache

import (
	"container/list"
	"sync"
)

// LRUCache is a thread-safe LRU cache for thumbnail data
type LRUCache struct {
	capacity int
	size     int64
	maxSize  int64 // max size in bytes
	items    map[string]*list.Element
	order    *list.List
	mu       sync.RWMutex
}

type cacheEntry struct {
	key  string
	data []byte
}

// NewLRUCache creates a new LRU cache with the specified capacity and max size in bytes
func NewLRUCache(capacity int, maxSizeBytes int64) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		maxSize:  maxSizeBytes,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

// Get retrieves an item from the cache
func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheEntry).data, true
	}
	return nil, false
}

// Set adds or updates an item in the cache
func (c *LRUCache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dataSize := int64(len(data))

	// If single item is larger than max size, don't cache it
	if dataSize > c.maxSize {
		return
	}

	// If key already exists, update it
	if elem, ok := c.items[key]; ok {
		oldEntry := elem.Value.(*cacheEntry)
		c.size -= int64(len(oldEntry.data))
		oldEntry.data = data
		c.size += dataSize
		c.order.MoveToFront(elem)
		return
	}

	// Evict items until we have space
	for c.order.Len() >= c.capacity || (c.size+dataSize > c.maxSize && c.order.Len() > 0) {
		c.evictOldest()
	}

	// Add new item
	entry := &cacheEntry{key: key, data: data}
	elem := c.order.PushFront(entry)
	c.items[key] = elem
	c.size += dataSize
}

// Delete removes an item from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
	c.size = 0
}

// Len returns the number of items in the cache
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.order.Len()
}

// Size returns the current size in bytes
func (c *LRUCache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

func (c *LRUCache) evictOldest() {
	elem := c.order.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *LRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	c.order.Remove(elem)
	delete(c.items, entry.key)
	c.size -= int64(len(entry.data))
}
