"""GUI app to generate/manage SSH keys for Git hosts."""

# pylint: disable=no-name-in-module,missing-function-docstring,missing-class-docstring,attribute-defined-outside-init,too-many-lines

import sys
import webbrowser
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import cast

from PySide6.QtCore import (
    QObject, QRunnable, QEasingCurve, Qt, QThreadPool, QUrl, Signal, QSize
)
from PySide6.QtCore import QPropertyAnimation, QTimer
from PySide6.QtGui import QDesktopServices, QGuiApplication, QIcon
from PySide6.QtWidgets import QProgressBar, QStyle
from PySide6.QtWidgets import (
    QApplication,
    QCheckBox,
    QFrame,
    QGroupBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QMainWindow,
    QMessageBox,
    QPushButton,
    QPlainTextEdit,
    QStackedWidget,
    QSplitter,
    QTabWidget,
    QToolButton,
    QVBoxLayout,
    QWidget,
)

from ssh_utils import (
    generate_key,
    list_ssh_keys,
    load_public_key,
    test_connections,
    KeyAlgorithm,
)


GITHUB_SSH_URL = "https://github.com/settings/ssh/new"
BITBUCKET_SSH_URL = "https://bitbucket.org/account/settings/ssh-keys/"


APP_STYLESHEET = """
QWidget { font-family: Segoe UI, Inter, Arial; font-size: 10.5pt; }
QMainWindow { background: #0f1115; }

QLabel#Title { font-size: 18pt; font-weight: 650; color: #f3f4f6; }
QLabel#Subtitle { color: #b8c0cc; }

QGroupBox {
    color: #e8ecf3;
    border: 1px solid #252a33;
    border-radius: 12px;
    margin-top: 10px;
    padding: 10px;
    background: #12151b;
}
QGroupBox::title {
    subcontrol-origin: margin;
    left: 10px;
    padding: 0 6px;
    color: #dbe3ef;
}

QLineEdit, QPlainTextEdit {
    background: #0c0f14;
    border: 1px solid #252a33;
    border-radius: 10px;
    padding: 8px;
    color: #e8ecf3;
    selection-background-color: #2b5cff;
}

QPlainTextEdit { font-family: Consolas, ui-monospace, monospace; font-size: 10pt; }

QPushButton {
    background: #1a2230;
    border: 1px solid #2a3342;
    border-radius: 10px;
    padding: 9px 12px;
    color: #f3f4f6;
}
QPushButton:hover { background: #212b3a; }
QPushButton:pressed { background: #151c27; }
QPushButton:disabled { color: #7b8594; background: #141821; border-color: #232936; }

QPushButton#Primary {
    background: #2b5cff;
    border-color: #2b5cff;
}
QPushButton#Primary:hover { background: #2551e6; }
QPushButton#Danger {
    background: #2a1618;
    border-color: #513034;
    color: #ffd5d8;
}
QPushButton#Danger:hover { background: #351b1e; }

QTabWidget::pane { border: 1px solid #252a33; border-radius: 12px; background: #12151b; }
QTabBar::tab {
    background: #12151b;
    border: 1px solid #252a33;
    padding: 8px 12px;
    border-top-left-radius: 10px;
    border-top-right-radius: 10px;
    color: #cfd6e2;
    margin-right: 6px;
}
QTabBar::tab:selected { background: #151a22; color: #f3f4f6; }

QFrame#Panel { background: transparent; }

QToolButton#Link {
    background: transparent;
    border: 1px solid transparent;
    color: #9bb7ff;
    padding: 6px 0;
    text-align: left;
}
QToolButton#Link:hover { color: #c7d6ff; }

QStatusBar { background: #0f1115; color: #b8c0cc; }

QFrame#BusyOverlay {
    background: rgba(15, 17, 21, 240);
    border-radius: 14px;
}

QPushButton#Cancel {
    background: #3d1f1e;
    border-color: #5a3230;
    color: #ffb4b0;
}
QPushButton#Cancel:hover { background: #4a2624; }

QToolButton#Toggle {
    background: #1a2230;
    border: 1px solid #2a3342;
    border-radius: 8px;
    padding: 6px 10px;
    color: #f3f4f6;
}
QToolButton#Toggle:hover { background: #212b3a; }
QToolButton#Toggle:checked {
    background: #2b5cff;
    border-color: #2b5cff;
}
"""


def _ts() -> str:
    return datetime.now().strftime("%H:%M:%S")


class _WorkerSignals(QObject):  # pylint: disable=too-few-public-methods
    finished = Signal(object)
    failed = Signal(str)


class _Worker(QRunnable):  # pylint: disable=too-few-public-methods
    def __init__(self, fn):
        super().__init__()
        self.fn = fn
        self.signals = _WorkerSignals()
        self._cancelled = False

    def cancel(self):
        self._cancelled = True

    def run(self):
        try:
            if not self._cancelled:
                result = self.fn()
                if not self._cancelled:
                    self.signals.finished.emit(result)
        except Exception as exc:  # noqa: BLE001  # pylint: disable=broad-exception-caught
            if not self._cancelled:
                self.signals.failed.emit(str(exc))


class SSHApp(QMainWindow):
    # pylint: disable=too-many-instance-attributes
    def __init__(self):
        super().__init__()
        self.setWindowTitle("SSH Key Setup")
        self.resize(980, 640)
        self._pool = QThreadPool.globalInstance()
        self._busy = False
        self._current_worker = None

        # Optional UI elements kept for backward compatibility with earlier layouts.
        # (Used by _handle_test_results; may remain None in the current UI.)
        self.lbl_test: QLabel | None = None

        # Key management state
        self._selected_key = None
        self._used_keys = set()  # Track keys added to Git hosts
        self._test_ok = False

        self._build_ui()
        self.refresh_state()

    def _build_ui(self):  # pylint: disable=too-many-statements,too-many-locals
        # QApplication.instance() is typed as Optional[QCoreApplication]; narrow for type checkers.
        qt_app = cast(QApplication, QApplication.instance())
        qt_app.setStyleSheet(APP_STYLESHEET)

        root = QWidget(self)
        self.setCentralWidget(root)
        outer = QVBoxLayout(root)
        outer.setContentsMargins(18, 18, 18, 14)
        outer.setSpacing(12)

        self._root = root

        # Clean header with title only
        header = QVBoxLayout()
        self.title = QLabel("SSH Key Setup")
        self.title.setObjectName("Title")
        header.addWidget(self.title)
        outer.addLayout(header)

        # Main content (tabs only, no sidebar)
        right = QTabWidget()
        outer.addWidget(right, 1)

        self.tabs = right

        guided_tab = self._build_guided_tab()
        right.addTab(guided_tab, "Guided setup")

        # Public key tab
        key_tab = QWidget()
        key_tab_layout = QVBoxLayout(key_tab)
        key_tab_layout.setContentsMargins(14, 14, 14, 14)
        key_tab_layout.setSpacing(10)

        row = QHBoxLayout()
        row.setSpacing(10)
        self.btn_copy = QPushButton("Copy public key")
        self.btn_copy.setObjectName("Primary")
        self.btn_open_ssh = QPushButton("Open .ssh folder")
        row.addWidget(self.btn_copy)
        row.addWidget(self.btn_open_ssh)
        row.addStretch(1)
        key_tab_layout.addLayout(row)

        self.public_key = QPlainTextEdit()
        self.public_key.setReadOnly(True)
        self.public_key.setPlaceholderText("Your public key will appear here after generation.")
        key_tab_layout.addWidget(self.public_key, 1)
        right.addTab(key_tab, "Public key")

        # Logs tab
        log_tab = QWidget()
        log_layout = QVBoxLayout(log_tab)
        log_layout.setContentsMargins(14, 14, 14, 14)
        log_layout.setSpacing(10)

        log_actions = QHBoxLayout()
        self.btn_clear_log = QPushButton("Clear log")
        log_actions.addWidget(self.btn_clear_log)
        log_actions.addStretch(1)
        log_layout.addLayout(log_actions)

        self.log = QPlainTextEdit()
        self.log.setReadOnly(True)
        log_layout.addWidget(self.log, 1)
        right.addTab(log_tab, "Logs")

        # Help tab
        help_tab = QWidget()
        help_layout = QVBoxLayout(help_tab)
        help_layout.setContentsMargins(14, 14, 14, 14)
        help_layout.setSpacing(10)
        help_text = QLabel(
            "<b>Recommended flow</b><br>"
            "1) Generate a key (optionally set a comment).<br>"
            "2) Start the SSH agent and add the key.<br>"
            "3) Copy the public key and add it to GitHub/Bitbucket.<br>"
            "4) Test the connection.<br><br>"
            "<b>Tip</b>: If you already created a key elsewhere, ensure it matches "
            "the path shown in Status."
        )
        help_text.setWordWrap(True)
        help_layout.addWidget(help_text)
        right.addTab(help_tab, "Help")

        # Default to the guided view
        right.setCurrentIndex(0)

        # Busy overlay (loader with cancel button)
        self._busy_overlay = QFrame(root)
        self._busy_overlay.setObjectName("BusyOverlay")
        self._busy_overlay.setVisible(False)
        self._busy_overlay.setFrameShape(QFrame.Shape.NoFrame)
        ov = QVBoxLayout(self._busy_overlay)
        ov.setContentsMargins(18, 18, 18, 18)
        ov.setSpacing(10)
        ov.addStretch(1)
        self._busy_label = QLabel("Working…")
        self._busy_label.setAlignment(Qt.AlignmentFlag.AlignHCenter)
        self._busy_label.setStyleSheet("color: #e8ecf3; font-weight: 600;")
        ov.addWidget(self._busy_label)
        self._busy_bar = QProgressBar()
        self._busy_bar.setRange(0, 0)
        self._busy_bar.setTextVisible(False)
        ov.addWidget(self._busy_bar)
        
        self._busy_cancel = QPushButton("Cancel")
        self._busy_cancel.setObjectName("Cancel")
        self._busy_cancel.clicked.connect(self._cancel_operation)
        ov.addWidget(self._busy_cancel)
        ov.addStretch(2)

        self.setStatusBar(self.statusBar())
        self.statusBar().showMessage("Ready")

        # Signals for tab content
        self.btn_copy.clicked.connect(self.on_copy)
        self.btn_open_ssh.clicked.connect(self.on_open_ssh_folder)
        self.btn_clear_log.clicked.connect(self.log.clear)

        self._update_overlay_geometry()

    def resizeEvent(self, event):  # noqa: N802  # pylint: disable=invalid-name
        super().resizeEvent(event)
        self._update_overlay_geometry()

    def _update_overlay_geometry(self):
        if hasattr(self, "_busy_overlay"):
            self._busy_overlay.setGeometry(self._root.rect())

    def _cancel_operation(self):
        if self._current_worker:
            self._current_worker.cancel()
            self._current_worker = None
            self._set_busy(False, "Cancelled")
            self.write("Operation cancelled by user")

    def _build_guided_tab(self) -> QWidget:  # pylint: disable=too-many-statements
        tab = QWidget()
        layout = QVBoxLayout(tab)
        layout.setContentsMargins(14, 14, 14, 14)
        layout.setSpacing(10)

        header = QGroupBox("Step-by-step setup")
        header_layout = QVBoxLayout(header)
        title = QLabel("Complete SSH setup for Git in five steps")
        title.setStyleSheet("font-weight: 600;")
        hint = QLabel(
            "Run each step in order. You can switch to the Public key tab at any time "
            "to view or copy the key."
        )
        hint.setWordWrap(True)
        header_layout.addWidget(title)
        header_layout.addWidget(hint)
        layout.addWidget(header)

        # Breadcrumb navigation
        self.guided_crumbs_row = QHBoxLayout()
        self.guided_crumbs_row.setSpacing(6)
        layout.addLayout(self.guided_crumbs_row)

        self._crumb_titles = [
            "Get started",
            "Generate key",
            "Manage keys",
            "Add to Git host",
            "Test",
        ]
        self._crumb_buttons = []
        for i, t in enumerate(self._crumb_titles):
            btn = QToolButton()
            btn.setObjectName("Link")
            btn.setText(t)
            btn.clicked.connect(lambda _=False, idx=i: self._guided_go_to(idx))
            self._crumb_buttons.append(btn)
            self.guided_crumbs_row.addWidget(btn)
            if i != len(self._crumb_titles) - 1:
                sep = QLabel("/")
                sep.setStyleSheet("color: #6b7280;")
                self.guided_crumbs_row.addWidget(sep)
        self.guided_crumbs_row.addStretch(1)

        self.guided_stack = QStackedWidget()
        layout.addWidget(self.guided_stack, 1)

        self.guided_stack.addWidget(self._guided_step_get_started())
        self.guided_stack.addWidget(self._guided_step_key())
        self.guided_stack.addWidget(self._guided_step_manage())
        self.guided_stack.addWidget(self._guided_step_publish())
        self.guided_stack.addWidget(self._guided_step_test())

        nav = QHBoxLayout()
        nav.setSpacing(10)
        self.guided_back = QPushButton("Back")
        self.guided_next = QPushButton("Next")
        self.guided_next.setObjectName("Primary")
        nav.addWidget(self.guided_back)
        nav.addWidget(self.guided_next)
        nav.addStretch(1)
        layout.addLayout(nav)

        self.guided_back.clicked.connect(self._guided_prev)
        self.guided_next.clicked.connect(self._guided_next)

        self._guided_update_nav()
        return tab

    def _guided_step_get_started(self) -> QWidget:
        w = QWidget()
        l = QVBoxLayout(w)
        l.setSpacing(10)

        title = QLabel("Get started")
        title.setStyleSheet("font-weight: 700; font-size: 13pt;")
        l.addWidget(title)

        body = QLabel(
            "This application helps you create and manage SSH keys for Git. "
            "You can generate keys with different algorithms, manage multiple keys, "
            "and track which ones you've added to Git hosts."
        )
        body.setWordWrap(True)
        l.addWidget(body)

        features = QLabel(
            "• Generate keys with Ed25519, RSA, or ECDSA algorithms\n"
            "• Create multiple keys with custom names\n"
            "• View and copy all your keys\n"
            "• Track which keys are in use"
        )
        features.setStyleSheet("color: #b8c0cc;")
        l.addWidget(features)

        self.guided_start_btn = QPushButton("Start setup")
        self.guided_start_btn.setObjectName("Primary")
        self.guided_start_btn.clicked.connect(lambda: self._guided_go_to(1))
        l.addWidget(self.guided_start_btn)

        l.addStretch(1)
        return w

    def _guided_step_key(self) -> QWidget:
        w = QWidget()
        l = QVBoxLayout(w)
        l.setSpacing(10)

        title = QLabel("Step 1: Create an SSH key")
        title.setStyleSheet("font-weight: 600;")
        l.addWidget(title)

        body = QLabel(
            "Generate a new SSH key with your preferred algorithm. "
            "You can create multiple keys with different names."
        )
        body.setWordWrap(True)
        l.addWidget(body)

        # Algorithm selection
        algo_label = QLabel("Algorithm:")
        algo_label.setStyleSheet("font-weight: 500;")
        l.addWidget(algo_label)
        
        algo_row = QHBoxLayout()
        algo_row.setSpacing(8)
        self.algo_ed25519 = QToolButton()
        self.algo_ed25519.setObjectName("Toggle")
        self.algo_ed25519.setText("Ed25519 (recommended)")
        self.algo_ed25519.setCheckable(True)
        self.algo_ed25519.setChecked(True)
        
        self.algo_rsa = QToolButton()
        self.algo_rsa.setObjectName("Toggle")
        self.algo_rsa.setText("RSA 4096")
        self.algo_rsa.setCheckable(True)
        
        self.algo_ecdsa = QToolButton()
        self.algo_ecdsa.setObjectName("Toggle")
        self.algo_ecdsa.setText("ECDSA")
        self.algo_ecdsa.setCheckable(True)
        
        # Make them mutually exclusive
        self.algo_ed25519.clicked.connect(lambda: self._select_algo("ed25519"))
        self.algo_rsa.clicked.connect(lambda: self._select_algo("rsa"))
        self.algo_ecdsa.clicked.connect(lambda: self._select_algo("ecdsa"))
        
        algo_row.addWidget(self.algo_ed25519)
        algo_row.addWidget(self.algo_rsa)
        algo_row.addWidget(self.algo_ecdsa)
        algo_row.addStretch(1)
        l.addLayout(algo_row)

        # Key name input
        name_label = QLabel("Key name:")
        name_label.setStyleSheet("font-weight: 500;")
        l.addWidget(name_label)
        
        self.guided_key_name = QLineEdit()
        self.guided_key_name.setPlaceholderText("e.g., id_ed25519, github_key, work_key")
        self.guided_key_name.setText("id_ed25519")
        l.addWidget(self.guided_key_name)

        # Comment input
        comment_label = QLabel("Comment (optional):")
        comment_label.setStyleSheet("font-weight: 500;")
        l.addWidget(comment_label)
        
        self.guided_comment = QLineEdit()
        self.guided_comment.setPlaceholderText("e.g., email@domain.com")
        l.addWidget(self.guided_comment)

        self.guided_force = QCheckBox("Overwrite if exists")
        self.guided_force.setToolTip("Replace existing key with the same name")
        l.addWidget(self.guided_force)

        self.guided_btn_key = QPushButton("Generate key")
        self.guided_btn_key.setObjectName("Primary")
        self.guided_btn_key.clicked.connect(self._guided_generate_key)
        l.addWidget(self.guided_btn_key)

        self.guided_key_status = QLabel("")
        self.guided_key_status.setWordWrap(True)
        self.guided_key_status.setStyleSheet("color: #10b981;")
        l.addWidget(self.guided_key_status)

        l.addStretch(1)
        return w
    
    def _select_algo(self, algo: str):
        """Handle algorithm toggle button selection."""
        self.algo_ed25519.setChecked(algo == "ed25519")
        self.algo_rsa.setChecked(algo == "rsa")
        self.algo_ecdsa.setChecked(algo == "ecdsa")
        
        # Update default key name based on algorithm
        if not self.guided_key_name.text() or self.guided_key_name.text().startswith("id_"):
            self.guided_key_name.setText(f"id_{algo}")

    def _guided_step_manage(self) -> QWidget:
        w = QWidget()
        l = QVBoxLayout(w)
        l.setSpacing(10)

        title = QLabel("Step 2: Manage your keys")
        title.setStyleSheet("font-weight: 600;")
        l.addWidget(title)

        body = QLabel(
            "View all your SSH keys, copy them, and track which ones you've added to Git hosts."
        )
        body.setWordWrap(True)
        l.addWidget(body)

        # Key list
        from PySide6.QtWidgets import QListWidget, QListWidgetItem
        self.key_list = QListWidget()
        self.key_list.setStyleSheet("""
            QListWidget {
                background: #0c0f14;
                border: 1px solid #252a33;
                border-radius: 10px;
                padding: 4px;
            }
            QListWidget::item {
                padding: 8px;
                border-radius: 6px;
                margin: 2px;
            }
            QListWidget::item:selected {
                background: #2b5cff;
                color: #ffffff;
            }
            QListWidget::item:hover {
                background: #1a2230;
            }
        """)
        self.key_list.itemSelectionChanged.connect(self._on_key_selected)
        l.addWidget(self.key_list)

        # Selected key info
        self.selected_key_label = QLabel("Select a key to view details")
        self.selected_key_label.setStyleSheet("color: #b8c0cc; font-style: italic;")
        self.selected_key_label.setWordWrap(True)
        l.addWidget(self.selected_key_label)

        # Action buttons
        btn_row = QHBoxLayout()
        btn_row.setSpacing(8)
        
        self.btn_copy_selected = QPushButton("Copy selected key")
        self.btn_copy_selected.setObjectName("Primary")
        self.btn_copy_selected.setEnabled(False)
        self.btn_copy_selected.clicked.connect(self._copy_selected_key)
        
        self.btn_mark_used = QPushButton("Mark as used")
        self.btn_mark_used.setEnabled(False)
        self.btn_mark_used.clicked.connect(self._mark_key_used)
        
        self.btn_refresh_keys = QPushButton("Refresh list")
        self.btn_refresh_keys.clicked.connect(self._refresh_key_list)
        
        btn_row.addWidget(self.btn_copy_selected)
        btn_row.addWidget(self.btn_mark_used)
        btn_row.addWidget(self.btn_refresh_keys)
        btn_row.addStretch(1)
        l.addLayout(btn_row)

        l.addStretch(1)
        return w

    def _guided_step_publish(self) -> QWidget:
        w = QWidget()
        l = QVBoxLayout(w)
        l.setSpacing(10)

        title = QLabel("Step 3: Add the public key to your Git host")
        title.setStyleSheet("font-weight: 600;")
        l.addWidget(title)

        body = QLabel(
            "Copy the public key from the Manage Keys step and add it to your Git host account. "
            "After saving it, mark the key as used and continue to test the connection."
        )
        body.setWordWrap(True)
        l.addWidget(body)

        row = QHBoxLayout()
        row.setSpacing(10)
        self.guided_btn_open_github = QPushButton("Open GitHub settings")
        self.guided_btn_open_bitbucket = QPushButton("Open Bitbucket settings")
        self.guided_btn_open_github.clicked.connect(lambda: webbrowser.open(GITHUB_SSH_URL))
        self.guided_btn_open_bitbucket.clicked.connect(lambda: webbrowser.open(BITBUCKET_SSH_URL))
        row.addWidget(self.guided_btn_open_github)
        row.addWidget(self.guided_btn_open_bitbucket)
        row.addStretch(1)
        l.addLayout(row)

        self.guided_publish_confirm = QCheckBox("I added the public key to my Git host")
        self.guided_publish_confirm.stateChanged.connect(lambda _: self._guided_update_nav())
        l.addWidget(self.guided_publish_confirm)

        l.addStretch(1)
        return w

    def _guided_step_test(self) -> QWidget:
        w = QWidget()
        l = QVBoxLayout(w)
        l.setSpacing(10)

        title = QLabel("Step 4: Test the SSH connection")
        title.setStyleSheet("font-weight: 600;")
        l.addWidget(title)

        body = QLabel(
            "This checks SSH authentication against GitHub and Bitbucket. "
            "See the Logs tab for the full output."
        )
        body.setWordWrap(True)
        l.addWidget(body)

        self.guided_test_status = QLabel("Status: not tested")
        self.guided_test_status.setWordWrap(True)
        l.addWidget(self.guided_test_status)

        self.guided_btn_test = QPushButton("Run connection test")
        self.guided_btn_test.setObjectName("Primary")
        self.guided_btn_test.clicked.connect(self._guided_test)
        l.addWidget(self.guided_btn_test)

        l.addStretch(1)
        return w

    def _guided_prev(self):
        idx = self.guided_stack.currentIndex()
        self.guided_stack.setCurrentIndex(max(0, idx - 1))
        self._guided_update_nav()

    def _guided_next(self):
        idx = self.guided_stack.currentIndex()
        self.guided_stack.setCurrentIndex(min(self.guided_stack.count() - 1, idx + 1))
        self._guided_update_nav()

    def _guided_go_to(self, idx: int):
        allowed = self._guided_max_index()
        self.guided_stack.setCurrentIndex(min(idx, allowed))
        self._guided_update_nav()

    def _guided_max_index(self) -> int:
        # Allow navigation through all steps
        if hasattr(self, "guided_publish_confirm") and self.guided_publish_confirm.isChecked():
            return 4
        return 3

    def _guided_update_nav(self):
        idx = self.guided_stack.currentIndex() if hasattr(self, "guided_stack") else 0
        total = self.guided_stack.count() if hasattr(self, "guided_stack") else 0
        if hasattr(self, "guided_back"):
            self.guided_back.setEnabled(idx > 0 and not self._busy)
            allow_next = idx < (total - 1) and not self._busy
            if idx == 4 and hasattr(self, "guided_publish_confirm"):
                allow_next = allow_next and self.guided_publish_confirm.isChecked()
            self.guided_next.setEnabled(allow_next)

        if hasattr(self, "_crumb_buttons"):
            allowed = self._guided_max_index()
            for i, btn in enumerate(self._crumb_buttons):
                btn.setEnabled(i <= allowed and not self._busy)
                if i == idx:
                    btn.setStyleSheet("color: #c7d6ff; font-weight: 650;")
                else:
                    btn.setStyleSheet("")

    def _guided_generate_key(self):
        key_name = self.guided_key_name.text().strip()
        if not key_name:
            QMessageBox.warning(self, "Invalid input", "Please enter a key name.")
            return
        
        # Get selected algorithm
        if self.algo_rsa.isChecked():
            algo = "rsa"
        elif self.algo_ecdsa.isChecked():
            algo = "ecdsa"
        else:
            algo = "ed25519"
        
        comment = self.guided_comment.text().strip() or None
        force = bool(self.guided_force.isChecked())

        if force:
            confirm = QMessageBox.warning(
                self,
                "Overwrite existing key?",
                f"This will overwrite the key '{key_name}' if it exists.\n\n"
                "Only do this if you understand the impact.",
                QMessageBox.StandardButton.Cancel | QMessageBox.StandardButton.Ok,
            )
            if confirm != QMessageBox.StandardButton.Ok:
                return

        def work():
            return generate_key(
                algorithm=algo,
                key_name=key_name,
                comment=comment,
                force=force
            )

        def done(msg):
            self.write(msg)
            self.guided_key_status.setText(f"✓ {msg}")
            self.refresh_state()
            self._refresh_key_list()
            
            # Animate the generate button
            self._animate_button(self.guided_btn_key)
            
            # Auto-advance after short delay
            QTimer.singleShot(800, lambda: self._guided_go_to(2))

        self._run_async("Generating key", work, done)

    def _guided_test(self):
        def work():
            key_name = None
            if getattr(self, "_selected_key", None):
                key_name = self._selected_key
            else:
                keys = list_ssh_keys()
                if keys:
                    key_name = keys[0].get("name")

            return test_connections(key_name)

        def done(results):
            any_ok = any(ok for ok, _ in results.values()) if results else False
            self._test_ok = bool(any_ok)
            self.guided_test_status.setText(
                f"Status: {'success' if any_ok else 'failed'}\nSee Logs for details."
            )
            self._handle_test_results(results)
            self._guided_update_nav()

        self._run_async("Testing SSH connections", work, done)

    def _handle_test_results(self, results):
        lines = []
        any_ok = False
        for host, (ok, msg) in results.items():
            lines.append(f"{host}: {'OK' if ok else 'FAIL'} — {msg}")
            any_ok = any_ok or ok

        # lbl_test existed in an earlier UI; keep this best-effort update for compatibility.
        result_text = f"Result: {'success' if any_ok else 'failed'}"
        if self.lbl_test is not None:
            self.lbl_test.setText(result_text)
        else:
            self.statusBar().showMessage(result_text)
        self.write("Connection test results:")
        for line in lines:
            self.write(f"  {line}")

        if not any_ok:
            QMessageBox.warning(
                self,
                "SSH test failed",
                "SSH authentication did not succeed.\n\n"
                "Make sure you've added the public key to your Git host. "
                "If you have multiple keys, select the key you added in 'Manage keys' "
                "so the test uses the right identity.",
            )

    def write(self, msg: str):
        self.log.appendPlainText(f"[{_ts()}] {msg}")

    def _set_busy(self, busy: bool, message: str | None = None):
        self._busy = busy
        
        # Only disable public key tab button
        if hasattr(self, "btn_copy"):
            self.btn_copy.setEnabled(not busy)
        if hasattr(self, "btn_open_ssh"):
            self.btn_open_ssh.setEnabled(not busy)
            
        if message:
            self.statusBar().showMessage(message)

        if hasattr(self, "guided_stack"):
            self._guided_update_nav()

        if hasattr(self, "_busy_overlay"):
            if busy:
                self._busy_label.setText(message or "Working…")
                self._busy_overlay.setVisible(True)
                self._busy_overlay.raise_()
            else:
                self._busy_overlay.setVisible(False)
                self._current_worker = None

    def _run_async(self, label: str, fn, on_done):
        if self._busy:
            return
        self._set_busy(True, f"{label}…")
        self.write(f"{label}…")

        worker = _Worker(fn)
        self._current_worker = worker
        worker.signals.finished.connect(lambda res: self._on_async_done(label, res, on_done))
        worker.signals.failed.connect(lambda err: self._on_async_fail(label, err))
        self._pool.start(worker)

    def _on_async_done(self, label: str, result, on_done):
        try:
            self.write(f"{label} done.")
            on_done(result)
        finally:
            self._set_busy(False, "Ready")

    def _on_async_fail(self, label: str, err: str):
        self._set_busy(False, "Ready")
        self.write(f"{label} failed: {err}")
        self._show_actionable_error(
            title="Operation failed",
            summary=f"{label} failed.",
            details=err,
        )

    def _show_actionable_error(
        self,
        title: str,
        summary: str,
        details: str | None = None,
        next_steps: str | None = None,
    ):
        box = QMessageBox(self)
        box.setIcon(QMessageBox.Icon.Critical)
        box.setWindowTitle(title)
        box.setText(summary)
        if next_steps:
            box.setInformativeText(next_steps)
        if details:
            box.setDetailedText(details)
        box.exec()



    def refresh_state(self):
        # Populate key list
        if hasattr(self, "key_list"):
            self._refresh_key_list()
        
        # Update public key tab with all keys
        keys = list_ssh_keys()
        if keys:
            all_keys_text = ""
            for key_info in keys:
                pub = load_public_key(key_info["name"])
                if pub:
                    used_marker = "[USED] " if key_info["name"] in self._used_keys else ""
                    all_keys_text += f"# {used_marker}{key_info['name']}\n{pub}\n\n"
            self.public_key.setPlainText(all_keys_text.strip())
        else:
            self.public_key.setPlainText("No keys found. Generate one first.")
        
        has_keys = bool(keys)
        self.btn_copy.setEnabled(has_keys and not self._busy)
        
        if hasattr(self, "guided_btn_key"):
            self.guided_btn_key.setEnabled(not self._busy)
            self.guided_btn_test.setEnabled(not self._busy)

        if hasattr(self, "guided_stack"):
            self._guided_update_nav()



    def _refresh_key_list(self):
        """Refresh the list of SSH keys."""
        if not hasattr(self, "key_list"):
            return

        # Import here so QListWidgetItem is always bound, even when there are no keys.
        from PySide6.QtWidgets import QListWidgetItem
        
        self.key_list.clear()
        keys = list_ssh_keys()
        
        for key_info in keys:
            key_name = key_info["name"]
            is_used = key_name in self._used_keys

            item = QListWidgetItem()
            
            status = "✓ Used" if is_used else "○ Not used"
            item.setText(f"{key_name}  [{status}]")
            item.setData(Qt.ItemDataRole.UserRole, key_info)
            
            self.key_list.addItem(item)
        
        if not keys:
            item = QListWidgetItem("No keys found. Generate one first.")
            item.setFlags(Qt.ItemFlag.NoItemFlags)
            self.key_list.addItem(item)
    
    def _on_key_selected(self):
        """Handle key selection from the list."""
        items = self.key_list.selectedItems()
        if not items:
            self.btn_copy_selected.setEnabled(False)
            self.btn_mark_used.setEnabled(False)
            self.selected_key_label.setText("Select a key to view details")
            self._selected_key = None
            return
        
        key_info = items[0].data(Qt.ItemDataRole.UserRole)
        if not key_info:
            return
        
        self._selected_key = key_info["name"]
        self.btn_copy_selected.setEnabled(True)
        self.btn_mark_used.setEnabled(True)
        
        pub_key = load_public_key(self._selected_key)
        if pub_key:
            preview = pub_key[:60] + "..." if len(pub_key) > 60 else pub_key
            self.selected_key_label.setText(f"Key: {self._selected_key}\n{preview}")
            self.selected_key_label.setStyleSheet("color: #e8ecf3;")
    
    def _copy_selected_key(self):
        """Copy the selected key to clipboard."""
        if not self._selected_key:
            return
        
        pub_key = load_public_key(self._selected_key)
        if not pub_key:
            QMessageBox.critical(self, "Error", "Public key not found.")
            return
        
        QGuiApplication.clipboard().setText(pub_key)
        self.statusBar().showMessage(f"Copied {self._selected_key}")
        self.write(f"Copied {self._selected_key} to clipboard")
        self._animate_button(self.btn_copy_selected)
    
    def _mark_key_used(self):
        """Mark the selected key as used."""
        if not self._selected_key:
            return
        
        self._used_keys.add(self._selected_key)
        self._refresh_key_list()
        self.statusBar().showMessage(f"Marked {self._selected_key} as used")
        self.write(f"Marked {self._selected_key} as used")
    
    def _animate_button(self, button: QPushButton):
        """Add a success animation to a button."""
        original_text = button.text()
        button.setText("✓ " + original_text)
        
        # Scale animation
        anim = QPropertyAnimation(button, b"minimumHeight", self)
        anim.setDuration(200)
        start_height = button.height()
        anim.setStartValue(start_height)
        anim.setEndValue(int(start_height * 1.08))
        anim.setEasingCurve(QEasingCurve.Type.OutCubic)
        anim.start()
        
        def reset():
            button.setText(original_text)
            button.setMinimumHeight(0)
        
        QTimer.singleShot(600, reset)

    def on_copy(self):
        """Copy key from the public key tab."""
        text = self.public_key.toPlainText().strip()
        if not text:
            QMessageBox.critical(self, "Error", "No public key to copy.")
            return
        
        QGuiApplication.clipboard().setText(text)
        self.statusBar().showMessage("Public key copied")
        self.write("Public key copied to clipboard")
        self._animate_button(self.btn_copy)





    def on_open_ssh_folder(self):
        # Open the .ssh folder
        try:
            ssh_dir = Path.home() / ".ssh"
            QDesktopServices.openUrl(QUrl.fromLocalFile(str(ssh_dir)))
        except Exception as exc:  # noqa: BLE001  # pylint: disable=broad-exception-caught
            QMessageBox.critical(self, "Open folder failed", str(exc))




if __name__ == "__main__":
    app = QApplication(sys.argv)
    win = SSHApp()
    win.show()
    sys.exit(app.exec())
