package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kubernetes-issues/pkg/stats"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	var duration *time.Duration
	var pagination *bool
	duration = flag.Duration("duration", 15*time.Minute, "duration for this remote test to run")
	pagination = flag.Bool("enable-pagination", false, "enable pagination or not (default 'false')")

	flag.Parse()

	lg, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer lg.Sync()

	rStats := stats.New(lg)
	defer rStats.Flush()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				lg.Info("context cancelled; exit program")
				return
			case s := <-sigs:
				lg.Info("received stop signal, exiting program", zap.Stringer("signal", s))
				cancel()
			}
		}
	}()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	if *pagination {
		err = paginatedList(ctx, lg, clientset, *duration, rStats)
	} else {
		err = unPagingatedList(ctx, lg, clientset, *duration, rStats)
	}
	if err != nil {
		lg.Warn("fail list pods", zap.Error(err))
	}
}

func paginatedList(ctx context.Context, lg *zap.Logger, clientset *kubernetes.Clientset, duration time.Duration, r *stats.RequestStats) (err error) {
	return err
}

func unPagingatedList(ctx context.Context, lg *zap.Logger, clientset *kubernetes.Clientset, duration time.Duration, r *stats.RequestStats) (err error) {
	stop := time.After(duration)
	for {
		select {
		case <-ctx.Done():
			lg.Info("context cancelled, exiting unpaginatedList")
			return nil
		case <-stop:
			lg.Info("test ends after duration", zap.Duration("duration", duration))
			return nil
		default:
		}

		start := time.Now()
		pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
		took := time.Since(start)
		if err != nil {
			switch {
			case errors.IsResourceExpired(err):
				lg.Warn("fail list pods: internal resource version expired", zap.Error(err))
			case errors.IsServerTimeout(err):
				lg.Warn("fail list pods: server timeout", zap.Duration("took", took), zap.Error(err))
			default:
				lg.Warn("fail list pods", zap.Error(err))
			}
			r.IncrementFailureCnt()
			continue
		}
		r.IncrementSuccessCnt()
		r.Add(float64(took / time.Second))
		lg.Info("list pod completed",
			zap.Int("cnt", r.GetSuccessCnt()),
			zap.Int("pod-number", len(pods.Items)),
			zap.Duration("took", took),
		)

		// not overload apiserver
		time.Sleep(2 * time.Second)
	}
}
