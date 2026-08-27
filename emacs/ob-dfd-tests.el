;;; ob-dfd-tests.el --- Tests for ob-dfd  -*- lexical-binding: t; -*-
;;; Commentary:
;; Run with emacs/run-tests.sh, which builds dfd and points DFD_BIN at it.
;;; Code:

(require 'ert)
(require 'cl-lib)
(require 'org)
(require 'ob-dfd)

(defvar ob-dfd-tests--dir nil "Scratch directory for one test.")

(defun ob-dfd-tests--render (header body)
  "Execute a dfd block with HEADER and BODY, returning the output file."
  (let* ((out (expand-file-name "out.svg" ob-dfd-tests--dir))
         (org-confirm-babel-evaluate nil)
         (org-babel-dfd-command (getenv "DFD_BIN")))
    (with-temp-buffer
      (org-mode)
      (insert (format "#+begin_src dfd :file %s %s\n%s\n#+end_src\n" out header body))
      (goto-char (point-min))
      (forward-line 1)  ; inside the block
      (org-babel-execute-src-block))
    out))

(defun ob-dfd-tests--contents (file)
  (with-temp-buffer
    (insert-file-contents file)
    (buffer-string)))

(defmacro ob-dfd-tests--with-dir (&rest body)
  (declare (indent 0))
  `(let ((ob-dfd-tests--dir (make-temp-file "ob-dfd" t)))
     (unwind-protect (progn ,@body)
       (delete-directory ob-dfd-tests--dir t))))

(ert-deftest ob-dfd-renders-a-block-to-a-file ()
  (ob-dfd-tests--with-dir
    (let ((out (ob-dfd-tests--render "" "[A]\n> x\n[B]")))
      (should (file-exists-p out))
      (let ((svg (ob-dfd-tests--contents out)))
        (should (string-match-p "<svg" svg))
        (should (string-match-p ">A</text>" svg))
        (should (string-match-p ">x</text>" svg))))))

(ert-deftest ob-dfd-passes-named-header-args-as-flags ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":theme plex :per-row 2" "[A]\n[B]\n[C]"))))
      ;; the plex canvas, so --theme reached the binary
      (should (string-match-p "#F2F4F6" svg))
      ;; three boxes over two rows, so --per-row did too
      (should (string-match-p "height=\"290\"" svg)))))

(ert-deftest ob-dfd-passes-cmdline-as-an-escape-hatch ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":cmdline \"--font-size 20\"" "[A]"))))
      (should (string-match-p "font-size=\"20\"" svg)))))

(ert-deftest ob-dfd-substitutes-declared-variables ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":var svc=\"workspace agent\" :var db=\"Registry\""
                                      "[Start $svc]\n    > id\n    |${db}|"))))
      (should (string-match-p ">Start workspace agent</text>" svg))
      (should (string-match-p ">Registry</text>" svg)))))

(ert-deftest ob-dfd-reports-parse-errors-with-body-line-numbers ()
  (ob-dfd-tests--with-dir
    (let ((out (ob-dfd-tests--render "" "[A]\nwat")))
      (should-not (file-exists-p out))
      (let ((buf (get-buffer org-babel-error-buffer-name)))
        (should buf)
        (with-current-buffer buf
          ;; line 2 of the block body, not of the org file
          (should (string-match-p "2: unrecognized line" (buffer-string))))))))


;;; The rules the end-to-end tests cannot pin down.

(defun ob-dfd-tests--found (&rest names)
  "Return a stub `executable-find' that only knows NAMES."
  (lambda (program &optional _remote) (car (member program names))))

(ert-deftest ob-dfd-prefers-an-installed-binary ()
  (let ((org-babel-dfd-command nil))
    (cl-letf (((symbol-function 'executable-find) (ob-dfd-tests--found "dfd" "go")))
      (should (equal (org-babel-dfd--command) '("dfd"))))))

(ert-deftest ob-dfd-falls-back-to-go-run ()
  (let ((org-babel-dfd-command nil))
    (cl-letf (((symbol-function 'executable-find) (ob-dfd-tests--found "go")))
      (let ((cmd (org-babel-dfd--command)))
        (should (equal (butlast cmd) '("go" "run")))
        ;; The command lives at cmd/dfd; the module root holds no main
        ;; package, so a bare module path cannot be run.
        (should (string-suffix-p "/cmd/dfd@latest" (car (last cmd))))))))

(ert-deftest ob-dfd-errors-when-neither-is-present ()
  (let ((org-babel-dfd-command nil))
    (cl-letf (((symbol-function 'executable-find) (ob-dfd-tests--found)))
      (should-error (org-babel-dfd--command)))))

(ert-deftest ob-dfd-honours-an-explicit-command ()
  (let ((org-babel-dfd-command "/opt/dfd"))
    (should (equal (org-babel-dfd--command) '("/opt/dfd"))))
  (let ((org-babel-dfd-command '("go" "run" "./cmd/dfd")))
    (should (equal (org-babel-dfd--command) '("go" "run" "./cmd/dfd")))))

(ert-deftest ob-dfd-leaves-dollars-alone-without-vars ()
  (should (equal (org-babel-expand-body:dfd "[Pay $5]" nil) "[Pay $5]")))

(ert-deftest ob-dfd-substitutes-both-reference-forms ()
  (let ((vars '((db . "Registry") (n . 3))))
    (should (equal (org-babel-dfd--substitute "[$db]" vars) "[Registry]"))
    (should (equal (org-babel-dfd--substitute "[${db}s]" vars) "[Registrys]"))
    (should (equal (org-babel-dfd--substitute "[n=$n]" vars) "[n=3]"))))

(ert-deftest ob-dfd-unescapes-a-literal-dollar ()
  (should (equal (org-babel-dfd--substitute "[Pay $$5]" '((db . "x"))) "[Pay $5]")))

(ert-deftest ob-dfd-rejects-an-undeclared-variable ()
  (should-error (org-babel-dfd--substitute "[$nope]" '((db . "x")))))

(provide 'ob-dfd-tests)
;;; ob-dfd-tests.el ends here

(ert-deftest ob-dfd-turns-numbering-on-from-a-header-arg ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":number yes" "{Client}\n> req\n[Handle]\n    > row\n    |Users|"))))
      (should (string-match-p ">1</text>" svg))
      (should (string-match-p ">D1 Users</text>" svg)))))

(ert-deftest ob-dfd-leaves-numbering-off-by-default ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render "" "[Handle]\n    > row\n    |Users|"))))
      (should-not (string-match-p ">1</text>" svg))
      (should (string-match-p ">Users</text>" svg)))))

(ert-deftest ob-dfd-treats-a-negative-switch-as-off ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":number no" "[Handle]"))))
      (should-not (string-match-p ">1</text>" svg)))))

(ert-deftest ob-dfd-passes-the-number-prefix ()
  (ob-dfd-tests--with-dir
    (let ((svg (ob-dfd-tests--contents
                (ob-dfd-tests--render ":number yes :number-prefix \"2.\"" "[Handle]"))))
      (should (string-match-p ">2\\.1</text>" svg)))))

(ert-deftest ob-dfd-reads-switches-as-emacs-does ()
  (should (org-babel-dfd--on-p "yes"))
  (should (org-babel-dfd--on-p "t"))
  (should (org-babel-dfd--on-p "true"))
  (should (org-babel-dfd--on-p t))
  (should-not (org-babel-dfd--on-p "no"))
  (should-not (org-babel-dfd--on-p "nil"))
  (should-not (org-babel-dfd--on-p nil))
  (should-not (org-babel-dfd--on-p "")))

(ert-deftest ob-dfd-go-fallback-is-runnable ()
  "The fallback must name a command that exists, not just a string.
A string comparison cannot tell a real package path from a wrong one."
  (skip-unless (executable-find "go"))
  (let ((cmd (let ((org-babel-dfd-command nil))
               (cl-letf (((symbol-function 'executable-find) (ob-dfd-tests--found "go")))
                 (org-babel-dfd--command)))))
    (ob-dfd-tests--with-dir
      (let ((src (expand-file-name "in.dfd" ob-dfd-tests--dir))
            (out (expand-file-name "out.svg" ob-dfd-tests--dir))
            (err (expand-file-name "err.txt" ob-dfd-tests--dir)))
        (with-temp-file src (insert "[A]\n"))
        (let* ((status (apply #'call-process (car cmd) nil (list nil err) nil
                              (append (cdr cmd) (list src "-o" out))))
               (stderr (with-temp-buffer (insert-file-contents err) (buffer-string))))
          (when (string-match-p "dial tcp\\|no such host\\|connection refused\\|i/o timeout\\|certificate" stderr)
            (ert-skip (concat "cannot reach the module proxy: " stderr)))
          (should (equal status 0))
          (should (file-exists-p out)))))))
