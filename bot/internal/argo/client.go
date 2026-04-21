package argo

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var applicationGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

type Client struct {
	dyn       dynamic.Interface
	namespace string
}

func New(namespace string) (*Client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	return &Client{dyn: dyn, namespace: namespace}, nil
}

type Result int

const (
	ResultOK Result = iota
	ResultFailed
	ResultTimeout
	ResultNotFound
)

func (r Result) String() string {
	switch r {
	case ResultOK:
		return "OK"
	case ResultFailed:
		return "FAILED"
	case ResultTimeout:
		return "TIMEOUT"
	case ResultNotFound:
		return "NOT_FOUND"
	}
	return "UNKNOWN"
}

// WaitForSync polls the Application until a reconciliation that finished after
// `since` reaches a terminal state. Returns ResultOK when Synced+Healthy,
// ResultFailed when the sync failed, ResultTimeout on timeout, ResultNotFound
// if the Application does not exist.
func (c *Client) WaitForSync(ctx context.Context, appName string, since time.Time, interval, timeout time.Duration) (Result, error) {
	deadline := time.Now().Add(timeout)
	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		res, terminal, err := c.checkOnce(ctx, appName, since)
		if err != nil {
			if errors.IsNotFound(err) {
				return ResultNotFound, nil
			}
			return ResultFailed, err
		}
		if terminal {
			return res, nil
		}
		if time.Now().After(deadline) {
			return ResultTimeout, nil
		}
		select {
		case <-ctx.Done():
			return ResultTimeout, ctx.Err()
		case <-tick.C:
		}
	}
}

func (c *Client) checkOnce(ctx context.Context, appName string, since time.Time) (Result, bool, error) {
	app, err := c.dyn.Resource(applicationGVR).Namespace(c.namespace).Get(ctx, appName, metav1.GetOptions{})
	if err != nil {
		return ResultFailed, false, err
	}

	phase, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "phase")
	finishedAtStr, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "finishedAt")

	var finishedAt time.Time
	if finishedAtStr != "" {
		finishedAt, _ = time.Parse(time.RFC3339, finishedAtStr)
	}

	// Si el último sync terminó antes de que arrancáramos, no es el nuestro:
	// seguimos esperando.
	if finishedAt.Before(since) {
		return ResultFailed, false, nil
	}

	switch phase {
	case "Failed", "Error":
		return ResultFailed, true, nil
	case "Succeeded":
		syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
		healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
		if syncStatus == "Synced" && healthStatus == "Healthy" {
			return ResultOK, true, nil
		}
		// Sync OK pero aún reconciliando health; seguimos polleando.
		return ResultFailed, false, nil
	}
	return ResultFailed, false, nil
}
