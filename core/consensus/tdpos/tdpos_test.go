package tdpos

import (
	//"fmt"
	//"math/big"

	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/xuperchain/xuperchain/core/common/config"
	"github.com/xuperchain/xuperchain/core/consensus"
	crypto_client "github.com/xuperchain/xuperchain/core/crypto/client"
	"github.com/xuperchain/xuperchain/core/ledger"
	"github.com/xuperchain/xuperchain/core/pb"
	"github.com/xuperchain/xuperchain/core/utxo"
	"github.com/xuperchain/xuperchain/core/utxo/txhash"
)

const (
	engine        = "default"
	bobAddress    = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	bobPubkey     = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571}`
	bobPrivateKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571,"D":29079635126530934056640915735344231956621504557963207107451663058887647996601}`
)

type fakeBlockChainHolder struct {
	Ledger     *ledger.Ledger
	UtxoVM     *utxo.UtxoVM
	B0         *pb.InternalBlock
	B1         *pb.InternalBlock
	B2         *pb.InternalBlock
	PrivateKey *ecdsa.PrivateKey
}

func generateTx(coinbase bool, utxo *utxo.UtxoVM) *pb.Transaction {
	txReq := &pb.TxData{}
	txReq.Bcname = "xuper"
	txReq.FromAddr = bobAddress
	txReq.FromPubkey = bobPubkey
	txReq.FromScrkey = bobPrivateKey
	txReq.Nonce = "nonce"
	txReq.Timestamp = time.Now().UnixNano()
	txReq.Account = []*pb.TxDataAccount{
		{Address: bobAddress, Amount: "1000000", FrozenHeight: int64(1000000)},
	}
	tx, _ := utxo.GenerateTx(txReq)
	return tx
}

func prepareBlockchain() *fakeBlockChainHolder {
	workSpace, _ := ioutil.TempDir("/tmp", "")
	os.RemoveAll(workSpace)
	defer os.RemoveAll(workSpace)
	// 准备账本
	l, _ := ledger.NewLedger(workSpace, nil, nil, engine, crypto_client.CryptoTypeDefault)
	cryptoClient, _ := crypto_client.CreateCryptoClient(crypto_client.CryptoTypeDefault)
	privateKey, _ := cryptoClient.GetEcdsaPrivateKeyFromJSON([]byte(bobPrivateKey))

	t1 := &pb.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &pb.TxOutput{Amount: []byte("1000000000000"), ToAddr: []byte(bobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte(`{"maxblocksize" : "128"}`)
	t1.Txid, _ = txhash.MakeTransactionID(t1)
	block0, _ := l.FormatRootBlock([]*pb.Transaction{t1})
	status := l.ConfirmBlock(block0, true)
	// 准备utxovm
	address, _ := hex.DecodeString(bobAddress)
	utxovm, _ := utxo.NewUtxoVM("xuper", l, workSpace, bobPrivateKey, bobPubkey, address, nil, false, "default", crypto_client.CryptoTypeDefault)
	utxovm.Play(block0.GetBlockid())
	fmt.Print(status)

	t2 := generateTx(false, utxovm)
	block1, _ := l.FormatFakeBlock([]*pb.Transaction{t2}, []byte(bobAddress), privateKey, time.Now().UnixNano(), 1, 1, block0.GetBlockid(), big.NewInt(0), 1)
	status = l.ConfirmBlock(block1, false)
	utxovm.Play(block1.GetBlockid())

	t3 := generateTx(false, utxovm)
	block2, _ := l.FormatFakeBlock([]*pb.Transaction{t3}, []byte(bobAddress), privateKey, time.Now().UnixNano(), 2, 2, block1.GetBlockid(), big.NewInt(0), 2)
	status = l.ConfirmBlock(block2, false)
	utxovm.Play(block2.GetBlockid())

	return &fakeBlockChainHolder{
		Ledger:     l,
		UtxoVM:     utxovm,
		B0:         block0,
		B1:         block1,
		B2:         block2,
		PrivateKey: privateKey,
	}
}

func prepareSingleCon(path string, ledger *ledger.Ledger, utxoVM *utxo.UtxoVM) *consensus.PluggableConsensus {
	cfg := config.NewNodeConfig()
	cfg.Miner.Keypath = path
	rootConfig := map[string]interface{}{
		"name": consensus.ConsensusTypeTdpos,
		"config": map[string]interface{}{
			"timestamp":            "1559021720000000000",
			"proposer_num":         "1",
			"period":               "3000",
			"alternate_interval":   "3000",
			"term_interval":        "6000",
			"block_num":            "20",
			"vote_unit_price":      "1",
			"init_proposer":        "{\"1\":[\"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN\"]}",
			"init_proposer_neturl": "{\"1\":[\"127.0.0.1:9001\"]}",
		},
	}
	plugCons, _ := consensus.NewPluggableConsensus(nil, cfg, "xuper", ledger, utxoVM, rootConfig, crypto_client.CryptoTypeDefault, nil)
	return plugCons
}

func TestInitBFT(t *testing.T) {
	//consensus_path := "../data/keys"
	fakeBlockChainHolder := prepareBlockchain()
	//plugCons := prepareSingleCon(consensus_path, fakeBlockChainHolder.Ledger, fakeBlockChainHolder.UtxoVM)
	tdpos := TDpos{
		height:         0,
		bcname:         "xuper",
		ledger:         fakeBlockChainHolder.Ledger,
		utxoVM:         fakeBlockChainHolder.UtxoVM,
		effectiveDelay: 0,
	}
	cfg := config.NewNodeConfig()
	tdpos.initBFT(cfg)
}

func close() {
	os.RemoveAll(workspace)
}
