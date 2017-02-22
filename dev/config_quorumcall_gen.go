// DO NOT EDIT. Generated by 'gorums' plugin for protoc-gen-go
// Source file to edit is: config_quorumcall_tmpl

package dev

import "golang.org/x/net/context"

// Read invokes a Read quorum call on configuration c
// and returns the result as a ReadReply.
func (c *Configuration) Read(ctx context.Context, args *ReadRequest) (*ReadReply, error) {
	return c.mgr.read(ctx, c, args)
}

// Write invokes a Write quorum call on configuration c
// and returns the result as a WriteReply.
func (c *Configuration) Write(ctx context.Context, args *State) (*WriteReply, error) {
	return c.mgr.write(ctx, c, args)
}
