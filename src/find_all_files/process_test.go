package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestProcessAllFiles(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		failOnKey      string
		wantWrites     map[string]string
		wantWriteOrder []string
		wantErr        string
	}{
		{
			name:  "Files found",
			input: []string{"file1", "file2"},
			wantWrites: map[string]string{
				"ALL_FILES": "file1,file2",
				"has_files": "true",
			},
			wantWriteOrder: []string{"ALL_FILES", "has_files"},
		},
		{
			name:  "No files found",
			input: []string{},
			wantWrites: map[string]string{
				"has_files": "false",
			},
			wantWriteOrder: []string{"has_files"},
		},
		{
			name:           "WriteOutput fails on ALL_FILES",
			input:          []string{"file1", "file2"},
			failOnKey:      "ALL_FILES",
			wantErr:        "cannot write ALL_FILES to GITHUB_OUTPUT",
			wantWriteOrder: []string{"ALL_FILES"},
		},
		{
			name:           "WriteOutput fails on has_files true",
			input:          []string{"file1", "file2"},
			failOnKey:      "has_files",
			wantErr:        "cannot write has_files to GITHUB_OUTPUT",
			wantWriteOrder: []string{"ALL_FILES", "has_files"},
			wantWrites: map[string]string{
				"ALL_FILES": "file1,file2",
			},
		},
		{
			name:           "WriteOutput fails on has_files false",
			input:          []string{},
			failOnKey:      "has_files",
			wantErr:        "cannot write has_files to GITHUB_OUTPUT",
			wantWriteOrder: []string{"has_files"},
		},
		{
			name:  "Nil input behaves like no files",
			input: nil,
			wantWrites: map[string]string{
				"has_files": "false",
			},
			wantWriteOrder: []string{"has_files"},
		},
		{
			name:  "Preserves input order in ALL_FILES",
			input: []string{"b.json", "a.json", "c.json"},
			wantWrites: map[string]string{
				"ALL_FILES": "b.json,a.json,c.json",
				"has_files": "true",
			},
			wantWriteOrder: []string{"ALL_FILES", "has_files"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writes := make(map[string]string)
			var order []string

			mockWrite := func(key, value string) bool {
				order = append(order, key)
				if tt.failOnKey == key {
					return false
				}
				writes[key] = value
				return true
			}

			err := processAllFiles(tt.input, mockWrite)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error mismatch. want substring=%q got=%q", tt.wantErr, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantWrites := tt.wantWrites
			if wantWrites == nil {
				wantWrites = map[string]string{}
			}

			if !reflect.DeepEqual(writes, wantWrites) {
				t.Fatalf("writes mismatch. want=%v got=%v", wantWrites, writes)
			}
			if !reflect.DeepEqual(order, tt.wantWriteOrder) {
				t.Fatalf("write order mismatch. want=%v got=%v", tt.wantWriteOrder, order)
			}
		})
	}
}
