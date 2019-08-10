#!/usr/bin/emacs --script
;; Copyright 2019  Datawire.
;;
;; term-render: Use (term-mode) to render ANSI escape sequences as
;; they'd appear in a terminal.
;;
;; Usage: term-render FILENAME

(load "term" nil t)

(defun sleep-forever ()
  (sleep-for 60)
  (sleep-forever))

(defun main ()
  (if (/= 1 (length command-line-args-left))
      (error "%s: expected exactly 1 (filename) argument, got %s"
             (file-name-nondirectory load-file-name)
             (length command-line-args-left)))

  (with-temp-buffer
    ;; Initialize the terminal
    (term-mode)
    (term-reset-size 24 80)
    ;; Override term.el's term-handle-exit
    (defun term-handle-exit (process-name msg)
      (princ (buffer-string))
      (kill-emacs))
    ;; Send the file to the terminal
    (term-exec (current-buffer) (buffer-name) "cat" nil (list "--" (car command-line-args-left)))
    ;; Wait for (term-handle-exit) to call (kill-emacs)
    (sleep-forever)))

(main)
