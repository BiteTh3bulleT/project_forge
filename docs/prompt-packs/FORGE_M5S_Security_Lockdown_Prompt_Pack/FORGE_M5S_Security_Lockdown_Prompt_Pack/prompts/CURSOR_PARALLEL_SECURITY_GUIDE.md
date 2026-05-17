# Cursor Parallel Security Guide

Worker A: API auth, bind validation, Dockerfile.  
Worker B: Approval authority.  
Worker C: Project-context import scope.  
Worker D: Job recovery/shutdown.  
Worker E: Windows process parity.  
Worker F: CI/docs/license/secrets.

Merge order: A, B, C, D, E, F.

Do not edit `routes.go`, `phase2.go`, or config files concurrently.
