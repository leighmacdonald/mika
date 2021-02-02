package cmd

import (
	"context"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/mika/client"
	"github.com/leighmacdonald/mika/consts"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/rpc"
	"github.com/leighmacdonald/mika/store"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"io/ioutil"
)

var (
	cl                    pb.MikaClient
	torrentFile           string
	torrentAddParams      = &pb.TorrentAddParams{}
	torrentInfoHashParams = &pb.InfoHashParam{}
)

func connectRPC(cmd *cobra.Command, args []string) error {
	c, err := client.New()
	if err != nil {
		return consts.ErrCannotConnect
	}
	cl = c
	return nil
}

func renderTorrents(torrents []*store.Torrent, title string) {
	t := defaultTable(title)
	t.AppendHeader(table.Row{"info_hash", "sn", "up_tot", "dn_tot", "en", "reason", "x_up", "x_dn"})
	for _, tor := range torrents {
		t.AppendRow(table.Row{
			tor.InfoHash, tor.Snatches, tor.Uploaded, tor.Downloaded,
			tor.IsEnabled, tor.Reason, tor.MultiUp, tor.MultiDn})
	}
	t.Render()
}

// torrentCmd represents torrent admin commands
var torrentCmd = &cobra.Command{
	Use:               "torrent",
	Short:             "torrent commands",
	Long:              `torrent commands`,
	Aliases:           []string{"t"},
	PersistentPreRunE: connectRPC,
}

// torrentAddCmd can be used to add torrents
var torrentAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a torrent to the tracker",
	Long:  `Add a torrent to the tracker`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := metainfo.LoadFromFile(torrentFile)
		if err != nil {
			return errors.Wrapf(err, "Failed to read torrent meta info")
		}
		b, err := ioutil.ReadFile(torrentFile)
		if err != nil {
			return err
		}
		var info metainfo.Info
		if err := bencode.Unmarshal(b, &info); err != nil {
			return err
		}
		if torrentAddParams.Title == "" {
			torrentAddParams.Title = info.Name
		}
		torrentAddParams.InfoHash = f.HashInfoBytes().Bytes()
		r, err2 := cl.TorrentAdd(context.Background(), torrentAddParams)
		if err2 != nil {
			log.Fatalf("Failed to add new torrent: %v", err2)
		}
		torrent := rpc.PBtoTorrent(r)
		renderTorrents([]*store.Torrent{torrent}, "Torrent added successfully")
		return nil
	},
}

// torrentGetCmd can be used to add torrents
var torrentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a torrent from the tracker",
	Long:  `Get a torrent from the tracker`,
	Run: func(cmd *cobra.Command, args []string) {
		if torrentInfoHashParams.InfoHashHex != "" {
			var ih store.InfoHash
			if err := store.InfoHashFromHex(&ih, torrentInfoHashParams.InfoHashHex); err != nil {
				log.Fatalf(err.Error())
				return
			}
			torrentInfoHashParams.InfoHash = ih.Bytes()
		} else {
			log.Fatalf("Must supply an infohash to delete")
			return
		}
		t, err := cl.TorrentGet(context.Background(), torrentInfoHashParams)
		if err != nil {
			log.Fatalf("Failed to get torrent")
			return
		}
		torrent := rpc.PBtoTorrent(t)
		renderTorrents([]*store.Torrent{torrent}, "Torrent Info")
	},
}

// torrentListCmd can be used to list torrents
var torrentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known torrents",
	Long:  `List known torrents`,
	Run: func(cmd *cobra.Command, args []string) {
		stream, err := cl.TorrentAll(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Fatalf("Failed to fetch torrents: %v", err)
			return
		}
		var torrents []*store.Torrent
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				break
			}
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			torrents = append(torrents, rpc.PBtoTorrent(in))
		}
		renderTorrents(torrents, "List of all torrents")
	},
}

// torrentDeleteCmd can be used to delete torrents
var torrentDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a torrent from the tracker",
	Long:    `Delete a torrent from the tracker`,
	Aliases: []string{"del"},
	Run: func(cmd *cobra.Command, args []string) {
		if torrentInfoHashParams.InfoHashHex != "" {
			var ih store.InfoHash
			if err := store.InfoHashFromHex(&ih, torrentInfoHashParams.InfoHashHex); err != nil {
				log.Fatalf(err.Error())
				return
			}
			torrentInfoHashParams.InfoHash = ih.Bytes()
		} else {
			log.Fatalf("Must supply an infohash to delete")
			return
		}
		_, err2 := cl.TorrentDelete(context.Background(), torrentInfoHashParams)
		if err2 != nil {
			log.Fatalf("Failed to add new role: %v", err2)
			return
		}
		log.Infof("Torrent deleted successfully")
	},
}

func init() {
	rootCmd.AddCommand(torrentCmd)
	torrentCmd.AddCommand(torrentListCmd)
	torrentCmd.AddCommand(torrentAddCmd)
	torrentCmd.AddCommand(torrentGetCmd)
	torrentCmd.AddCommand(torrentDeleteCmd)

	torrentGetCmd.Flags().StringVarP(&torrentInfoHashParams.InfoHashHex, "infohash", "i", "", "infohash of the torrent")

	torrentDeleteCmd.Flags().StringVarP(&torrentInfoHashParams.InfoHashHex, "infohash", "i", "", "infohash of the torrent")

	torrentAddCmd.Flags().StringVarP(&torrentFile, "file", "f", "", "Torrent file to add")
	torrentAddCmd.Flags().StringVarP(&torrentAddParams.Title, "name", "n", "", "Name of the torrent")
	torrentAddCmd.Flags().Float64VarP(&torrentAddParams.MultiUp, "multi_up", "U", 1.0, "Upload multiplier")
	torrentAddCmd.Flags().Float64VarP(&torrentAddParams.MultiDn, "multi_dn", "D", 1.0, "Download multiplier")
}
