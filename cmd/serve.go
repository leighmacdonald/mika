package cmd

import (
	"context"
	"github.com/leighmacdonald/mika/config"
	pb "github.com/leighmacdonald/mika/proto"
	"github.com/leighmacdonald/mika/rpc"
	"github.com/leighmacdonald/mika/tracker"
	"github.com/leighmacdonald/mika/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"net"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the tracker and serve requests",
	Long:  `Start the tracker and serve requests`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		tracker.Init()

		btOpts := tracker.DefaultHTTPOpts()
		btOpts.ListenAddr = config.Tracker.Listen
		btOpts.UseTLS = config.Tracker.TLS
		btOpts.Handler = tracker.NewBitTorrentHandler()
		btServer := tracker.NewHTTPServer(btOpts)

		go tracker.PeerReaper(ctx)
		go tracker.StatWorker(ctx)

		lis, err := net.Listen("tcp", config.API.Listen)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		var rpcOpts []grpc.ServerOption
		//if *tls {
		//	if *certFile == "" {
		//		*certFile = data.Path("x509/server_cert.pem")
		//	}
		//	if *keyFile == "" {
		//		*keyFile = data.Path("x509/server_key.pem")
		//	}
		//	creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		//	if err != nil {
		//		log.Fatalf("Failed to generate credentials %v", err)
		//	}
		//	opts = []grpc.ServerOption{grpc.Creds(creds)}
		//}
		grpcServer := grpc.NewServer(rpcOpts...)
		pb.RegisterMikaServer(grpcServer, &rpc.MikaService{})
		go func() {
			log.Infof("Starting gRPC service")
			if errRpc := grpcServer.Serve(lis); errRpc != nil {
				log.Errorf("gRPC error: %v", errRpc)
			}
		}()

		go func() {
			log.Infof("Starting tracker service")
			if errRpc := btServer.ListenAndServe(); errRpc != nil {
				log.Errorf("HTTP error: %v", errRpc)
			}
		}()

		util.WaitForSignal(ctx, func(ctx context.Context) error {
			if err := btServer.Shutdown(ctx); err != nil {
				log.Fatalf("Error closing servers gracefully; %s", err)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
