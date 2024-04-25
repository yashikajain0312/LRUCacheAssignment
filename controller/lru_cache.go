package main

import (
    "container/list"
    "net/http"
    "sync"
    "time"
	"fmt"

    "github.com/gin-gonic/gin"
)

// cacheEntry represents an entry in the LRU cache.
type cacheEntry struct {
    key        string
    value      interface{}
    expiration time.Time
}

// LRUCache represents the LRU cache.
type LRUCache struct {
    capacity int
    cache    map[string]*list.Element
    list     *list.List
    mutex    sync.Mutex
}

// Get retrieves the value associated with the given key from the cache.
func (c *LRUCache) Get(key string) interface{} {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if element, ok := c.cache[key]; ok {
        entry := element.Value.(*cacheEntry)
        if entry.expiration.After(time.Now()) {
            c.list.MoveToFront(element)
            return entry.value
        }
        // If entry has expired, delete it from cache
        delete(c.cache, key)
        c.list.Remove(element)
    }
    return nil
}

// Set inserts or updates a key-value pair in the cache.
func (c *LRUCache) Set(key string, value interface{}, expiration time.Duration) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if element, ok := c.cache[key]; ok {
        c.list.MoveToFront(element)
        entry := element.Value.(*cacheEntry)
        entry.value = value
        entry.expiration = time.Now().Add(expiration)
    } else {
        entry := &cacheEntry{
            key:        key,
            value:      value,
            expiration: time.Now().Add(expiration),
        }
        element := c.list.PushFront(entry)
        c.cache[key] = element
        if len(c.cache) > c.capacity {
            // Remove least recently used entry if capacity exceeded
            delete(c.cache, c.list.Back().Value.(*cacheEntry).key)
            c.list.Remove(c.list.Back())
        }
    }
}

// Function to clear the entire cache
func (c *LRUCache) ClearCache() {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    c.cache = make(map[string]*list.Element)
    c.list.Init()
}

// Function to get cache state and remove expired entries
func (c *LRUCache) GetCacheState() []cacheEntry {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    // Create a slice to store non-expired cache entries
    nonExpiredEntries := make([]cacheEntry, 0, len(c.cache))

    // Iterate over cache entries
    for _, element := range c.cache {
        entry := element.Value.(*cacheEntry)

        // Check if entry has expired
        if entry.expiration.After(time.Now()) {
            // If not expired, include in cache state
            nonExpiredEntries = append(nonExpiredEntries, *entry)
        } else {
            delete(c.cache, entry.key)
            c.list.Remove(element)
        }
    }

    fmt.Println("cacheState", nonExpiredEntries)
    return nonExpiredEntries
}


func main() {
    // Initialize the LRU cache
    cache := &LRUCache{
        capacity: 1000, // adjust capacity as needed
        cache:    make(map[string]*list.Element),
        list:     list.New(),
    }

    // Initialize Gin router
    router := gin.Default()

    // Define API endpoints
    router.GET("/cache/:key", func(c *gin.Context) {
        key := c.Param("key")
        value := cache.Get(key)
        if value != nil {
            c.JSON(http.StatusOK, gin.H{"value": value})
        } else {
            c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
        }
    })

    router.POST("/cache/:key", func(c *gin.Context) {
        key := c.Param("key")
        var data struct {
            Value      interface{} `json:"value"`
            Expiration int         `json:"expiration"`
        }
        if err := c.BindJSON(&data); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        cache.Set(key, data.Value, time.Duration(data.Expiration)*time.Second)
        c.Status(http.StatusOK)
    })

    // Define API endpoint for clearing the cache
    router.DELETE("/cache", func(c *gin.Context) {
      	cache.ClearCache()
      	c.Status(http.StatusOK)
    })

	type CacheEntryResponse struct {
		Key        string      `json:"key"`
		Value      interface{} `json:"value"`
		Expiration time.Time   `json:"expiration"`
	}

	router.GET("/cache-state", func(c *gin.Context) {
        cacheState := cache.GetCacheState()
		fmt.Println("cacheStateeee", cacheState)
		// Convert cache state into cache entry responses
		var cacheStateResponse []CacheEntryResponse
		for _, entry := range cacheState {
			cacheStateResponse = append(cacheStateResponse, CacheEntryResponse{
				Key:        entry.key,
				Value:      entry.value,
				Expiration: entry.expiration,
			})
		}

        c.JSON(http.StatusOK, cacheStateResponse)
    })

    // Run the server
    if err := router.Run(":3000"); err != nil {
        panic(err)
    }
}
