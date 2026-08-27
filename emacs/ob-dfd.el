;;; ob-dfd.el --- Org babel support for dfd diagrams  -*- lexical-binding: t; -*-

;; Author: Marcin Bilski
;; Keywords: languages, tools, org
;; Package-Requires: ((emacs "26.1"))
;; URL: https://github.com/bilus/dfd

;;; Commentary:

;; Render dfd data-flow diagrams from org source blocks:
;;
;;   #+begin_src dfd :file flow.svg :theme plex
;;   [Start process]
;;   > foo
;;   [Finish]
;;   #+end_src
;;
;; The block body is fed to the dfd binary on stdin, so parse errors
;; report the line within the block.  :file is required and its
;; extension chooses SVG or PNG.

;;; Code:

(require 'ob)

(defcustom org-babel-dfd-command nil
  "How to run dfd, or nil to look for it.

Nil first tries a `dfd' binary on PATH, then `go run' against the
published module, which needs Go but no install.  A string names a
program; a list gives a program and fixed arguments."
  :group 'org-babel
  :type '(choice (const :tag "Detect automatically" nil)
                 (string :tag "Program")
                 (repeat :tag "Program and arguments" string)))

(defvar org-babel-default-header-args:dfd
  '((:results . "file") (:exports . "results"))
  "Default header arguments for dfd source blocks.")

(defconst org-babel-dfd--flags
  '((:theme . "--theme")
    (:per-row . "--per-row")
    (:box . "--box")
    (:font-size . "--font-size")
    (:scale . "--scale")
    (:format . "--format")
    (:number-prefix . "--number-prefix"))
  "Header argument to the dfd flag it sets, for flags taking a value.
Adding a flag to dfd needs one line here, and nothing else.")

(defconst org-babel-dfd--switches
  '((:number . "--number"))
  "Header argument to the dfd flag it turns on, for flags taking none.")

(defconst org-babel-dfd--var-re
  "\\$\\(?:\\(\\$\\)\\|{\\([^}\n]+\\)}\\|\\([A-Za-z_][A-Za-z0-9_-]*\\)\\)"
  "A `$$' escape, a `${name}' reference, or a bare `$name' one.")

(defun org-babel-dfd--command ()
  "Return the program and any fixed arguments, as a list of strings."
  (cond
   ((stringp org-babel-dfd-command) (list org-babel-dfd-command))
   ((consp org-babel-dfd-command) org-babel-dfd-command)
   ((executable-find "dfd") (list "dfd"))
   ((executable-find "go") (list "go" "run" "github.com/bilus/dfd/cmd/dfd@latest"))
   (t (error "dfd: no dfd binary and no go on PATH; install dfd, or set org-babel-dfd-command"))))

(defun org-babel-dfd--on-p (value)
  "Is VALUE an org header argument meaning yes?"
  (and value
       (if (stringp value)
           (member (downcase value) '("yes" "t" "true"))
         t)
       t))

(defun org-babel-dfd--value (name vars)
  "Look NAME up in VARS, an alist from `org-babel--get-vars'."
  (let ((cell (assq (intern name) vars)))
    (unless cell
      (error "dfd: no :var named %s (declared: %s)" name
             (if vars
                 (mapconcat (lambda (v) (symbol-name (car v))) vars ", ")
               "none")))
    (format "%s" (cdr cell))))

(defun org-babel-dfd--substitute (body vars)
  "Replace variable references in BODY with their values from VARS."
  (let ((out "") (start 0))
    (while (string-match org-babel-dfd--var-re body start)
      (setq out (concat out (substring body start (match-beginning 0))))
      (let ((escaped (match-string 1 body))
            (name (or (match-string 2 body) (match-string 3 body))))
        (setq out (concat out (if escaped "$" (org-babel-dfd--value name vars)))))
      (setq start (match-end 0)))
    (concat out (substring body start))))

(defun org-babel-expand-body:dfd (body params)
  "Substitute the :var values of PARAMS into BODY.

Only blocks that declare a variable are expanded, so `$' stays an
ordinary character in every other diagram."
  (let ((vars (org-babel--get-vars params)))
    (if vars (org-babel-dfd--substitute body vars) body)))

(defun org-babel-dfd--args (params)
  "Return the dfd flags PARAMS asks for."
  (let (args)
    (pcase-dolist (`(,key . ,flag) org-babel-dfd--flags)
      (let ((value (cdr (assq key params))))
        (when (and value (not (equal value "")))
          (setq args (append args (list flag (format "%s" value)))))))
    (pcase-dolist (`(,key . ,flag) org-babel-dfd--switches)
      (when (org-babel-dfd--on-p (cdr (assq key params)))
        (setq args (append args (list flag)))))
    (let ((extra (cdr (assq :cmdline params))))
      (when (and extra (not (equal extra "")))
        (setq args (append args (split-string-and-unquote (format "%s" extra))))))
    args))

(defun org-babel-execute:dfd (body params)
  "Render BODY as a diagram, with PARAMS supplying the header arguments.
Returns nil so that org links to the :file it wrote."
  (let ((file (cdr (assq :file params))))
    (unless file
      (error "dfd: blocks need a :file header argument, for example :file flow.svg"))
    (org-babel-eval
     (mapconcat #'shell-quote-argument
                (append (org-babel-dfd--command)
                        (org-babel-dfd--args params)
                        (list "-o" (expand-file-name file)))
                " ")
     (org-babel-expand-body:dfd body params)))
  nil)

(add-to-list 'org-babel-tangle-lang-exts '("dfd" . "dfd"))

(provide 'ob-dfd)
;;; ob-dfd.el ends here
