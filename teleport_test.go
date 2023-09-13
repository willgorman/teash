package main

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func Test_lsNodesJson(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lsNodesJson()
			spew.Dump(got, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("lsNodesJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("lsNodesJson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTeleport_GetNodes(t *testing.T) {
	type fields struct {
		nodes Nodes
	}
	type args struct {
		refresh bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Nodes
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Teleport{
				nodes: tt.fields.nodes,
			}
			got, err := tr.GetNodes(tt.args.refresh)
			if (err != nil) != tt.wantErr {
				t.Errorf("Teleport.GetNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Teleport.GetNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}
