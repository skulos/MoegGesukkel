package persistence

import (
	// "github.com/faabiosr/cachego"
	// "github.com/faabiosr/cachego/file"
	"context"
	"crypto/sha256"
	"os"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/jellydator/ttlcache/v3"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
)

var log = flogging.MustGetLogger("PERSISTENCE")

var cache *ttlcache.Cache[string, string]

func base58Encode(input []byte) []byte {
	log.Info("Encoding the input in => base58Encode")

	encode := base58.Encode(input)

	return []byte(encode)
}

func base58Decode(input []byte) []byte {
	log.Info("Decoding the input in => base58Decode")
	decode, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Panic(err)
	}
	return decode
}

func generateToken() string {
	log.Info("Generating Token")

	epoch := strconv.Itoa(int(time.Now().Unix()))
	didHash := sha256.Sum256([]byte(epoch))

	hasher := ripemd160.New()
	hasher.Write(didHash[:])
	hashedDid := hasher.Sum(nil)

	precheck := sha256.Sum256(hashedDid)
	checksum := sha256.Sum256(precheck[:])

	finalHash := append(hashedDid, checksum[:]...)
	address := base58Encode(finalHash)
	token := string(address)

	log.Info("Token: ", token)
	return token
}

func Init() {
	log.Info("Running the Init process")

	cache = ttlcache.New(
		ttlcache.WithTTL[string, string](30 * time.Minute),
	)

	cache.OnEviction(func(_ context.Context, _ ttlcache.EvictionReason, item *ttlcache.Item[string, string]) {
		path := item.Value()
		log.Info("Removing ", item.Key(), " @ ", path, " # ", item.TTL())
		os.Remove(path)
	})

	go cache.Start()
}

func UploadToCache(path string, ttl time.Duration) string {
	log.Info("UploadToCache: path = ", path, " and time = ", ttl)
	token := generateToken()

	log.Info("Putting token into cache and starting TTL")
	cache.Set(token, path, ttl)

	return token
}

func DownloadFromCache(token string) string {
	log.Info("DownloadFromCache: ", token)

	item := cache.Get(token).Value()
	log.Info("Getting token's value = ", item)

	return item
}
