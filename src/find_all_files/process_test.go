package main

import (
	"reflect"
	"testing"
)

func TestProcessAllFiles(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		failOnKey      string
		wantWrites     map[string]string
		wantWriteOrder []string
		wantPanic      bool
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
			name:      "WriteOutput fails on ALL_FILES",
			input:     []string{"file1", "file2"},
			failOnKey: "ALL_FILES",
			wantPanic: true,
		},
		{
			name:      "WriteOutput fails on has_files true",
			input:     []string{"file1", "file2"},
			failOnKey: "has_files",
			wantPanic: true,
		},
		{
			name:      "WriteOutput fails on has_files false",
			input:     []string{},
			failOnKey: "has_files",
			wantPanic: true,
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

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatalf("expected panic but got none")
					}
				}()
			}

			processAllFiles(tt.input, mockWrite)

			if tt.wantPanic {
				return
			}

			if !reflect.DeepEqual(writes, tt.wantWrites) {
				t.Fatalf("writes mismatch. want=%v got=%v", tt.wantWrites, writes)
			}
			if !reflect.DeepEqual(order, tt.wantWriteOrder) {
				t.Fatalf("write order mismatch. want=%v got=%v", tt.wantWriteOrder, order)
			}
		})
	}
}
