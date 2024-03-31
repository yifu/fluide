package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackpal/bencode-go"
	"golang.org/x/sys/unix"
)

// The port this application is listening to:
const Port = 6881

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

func (bInfo bencodeInfo) String() string {
	result := ""
	//result += fmt.Sprint("pieces: ", bInfo.Pieces, " ")
	result += fmt.Sprint("pieces len: ", bInfo.PieceLength, " ")
	result += fmt.Sprint("len: ", bInfo.Length, " ")
	result += fmt.Sprint("name: ", bInfo.Name, " ")
	return result
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func (bTorrent bencodeTorrent) String() string {
	result := ""
	result += fmt.Sprint("announce: ", bTorrent.Announce, " ")
	result += fmt.Sprint("Info: [", bTorrent.Info, "]")
	return result
}

type bencodeTrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (trackerResp bencodeTrackerResp) String() string {
	result := ""
	result += fmt.Sprint("interval: ", trackerResp.Interval, " ")
	result += fmt.Sprint("Peers: ", trackerResp.Peers, " ")
	return result
}

type bencodePeer struct {
	PeerId string `bencode:"peer_id"`
	Ip     string `bencode:"ip"`
	Port   int    `bencode:"port"`
}

func (peer bencodePeer) String() string {
	result := ""
	result += fmt.Sprint("PeerId: ", peer.PeerId, " ")
	result += fmt.Sprint("Ip: ", peer.Ip, " ")
	result += fmt.Sprint("Port: ", peer.Port, " ")
	return result
}

// Peer encodes connection information for a peer
type Peer struct {
	IP   net.IP
	Port uint16
}

// Unmarshal parses peer IP addresses and ports from a buffer
func Unmarshal(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 for IP, 2 for port
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		err := fmt.Errorf("received malformed peers")
		return nil, err
	}
	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	return peers, nil
}

// type TorrentConnection struct {
// 	conn                                                     net.Conn
// 	peer                                                     Peer
// 	am_choking, am_interested, peer_choking, peer_interested bool
// }

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("Usage: ", os.Args[0], " <torrent file>")
	}

	// Open the file and Unmarshall it:
	fn := os.Args[1]
	f, err := os.Open(fn)
	if err != nil {
		log.Fatalln(err)
	}

	data := bencodeTorrent{}
	err = bencode.Unmarshal(f, &data)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(data)

	// Marshall the info part, in order to compute its sha-1 value:
	var buf bytes.Buffer
	err = bencode.Marshal(&buf, data.Info)
	if err != nil {
		log.Fatalln(err)
	}
	info_hash := sha1.Sum(buf.Bytes())
	fmt.Println("info hash: ", info_hash)

	// Generate a random peer id:
	peerid := make([]byte, 20)
	_, err = rand.Read(peerid)
	if err != nil {
		log.Fatalf("error while generating peer id: %s", err)
	}
	fmt.Println("peer id: ", peerid)

	// Compute the correct URL request to send to the tracker:
	base, err := url.Parse(data.Announce)
	if err != nil {
		log.Fatalln(err)
	}
	params := url.Values{
		"info_hash":  []string{string(info_hash[:])},
		"peer_id":    []string{string(peerid)},
		"port":       []string{strconv.Itoa(int(Port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(data.Info.Length)},
	}
	base.RawQuery = params.Encode()
	fmt.Println("url: ", base.String())

	// Create the HTTP client and connect to the tracker:
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(base.String())
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	fmt.Println("Response: ", resp)

	// Read all the response body, just to print it:
	// buffer, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println("Response body (after read): ", buffer)

	// Decode the body of the response:
	// d, err := bencode.Decode(resp.Body)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	//fmt.Println("decoded body: ", d)

	// Unmarshall the response body:
	trackerResp := bencodeTrackerResp{}
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		log.Fatalln(err)
	}
	//fmt.Println("tracker resp: ", trackerResp)

	// Check if the tracker Peers response is in compact format or not.
	log.Println("len(trackerResp.Peers)", len(trackerResp.Peers), ", trackerResp.Peers[0]", string(trackerResp.Peers[0]))
	var peers []Peer
	if len(trackerResp.Peers) > 0 && trackerResp.Peers[0] == 'd' {
		log.Println("We are in a dictionary format!")
		// The reponse is a dictionnary, as a consequence it is not in compact format.
		rdr := strings.NewReader(trackerResp.Peers)
		peer := bencodePeer{}
		err = bencode.Unmarshal(rdr, &peer)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(peer)
	} else {
		log.Println("Not in a dictionnary format! (i.e compact format)")
		peers, err = Unmarshal([]byte(trackerResp.Peers))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(peers)
	}

	// Create the recipient file:
	recipientFn := data.Info.Name
	recipientFn += ".fluide"

	// const oneByteLen = 8
	// piecesNbr := data.Info.Length / data.Info.PieceLength
	// piecesLen := piecesNbr / 8
	fileLen := data.Info.Length
	// totalLen := oneByteLen + piecesLen + fileLen
	totalLen := fileLen
	log.Println("total len is ", totalLen, ".")

	// TODO: try to open the file first,
	// and when the call returns an error "does not exist" then create the file!
	f, err = os.Create(recipientFn)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	unix.Ftruncate(int(f.Fd()), int64(totalLen))

	var store *unix.Fstore_t = &unix.Fstore_t{
		Flags:   unix.F_ALLOCATECONTIG | unix.F_ALLOCATEALL,
		Posmode: unix.F_PEOFPOSMODE,
		Offset:  0,
		Length:  int64(totalLen),
		// Bytesalloc: 0,
	}
	err = unix.FcntlFstore(f.Fd(), unix.F_PREALLOCATE, store)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("File ", recipientFn, " create with size of ", store.Bytesalloc, ".")

	var wg sync.WaitGroup
	if peers == nil {

	} else {
		for _, p := range peers {
			wg.Add(1)
			go connectToPeer(p, &wg)
		}
		wg.Wait()
		log.Println("Exit.")
	}
}

func connectToPeer(p Peer, wg *sync.WaitGroup) {
	log.Println("Connect to peer ", p)
	defer wg.Done()
}
