"""GUI app to generate and manage SSH keys for Git hosts."""

# pylint: disable=no-name-in-module,missing-function-docstring,missing-class-docstring,attribute-defined-outside-init,too-many-lines

import re
import sys
import webbrowser
import json
from datetime import datetime
from pathlib import Path
from typing import Any, cast

from PySide6.QtCore import QObject, QRunnable, Qt, QThreadPool, Signal
from PySide6.QtGui import QGuiApplication
from PySide6.QtWidgets import (
    QApplication,
    QCheckBox,
    QComboBox,
    QFrame,
    QFormLayout,
    QGroupBox,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QListWidget,
    QListWidgetItem,
    QMainWindow,
    QMessageBox,
    QPlainTextEdit,
    QProgressBar,
    QPushButton,
    QSplitter,
    QStackedWidget,
    QVBoxLayout,
    QWidget,
)

from ssh_utils import (
    KeyAlgorithm,
    add_key_to_agent,
    generate_key,
    get_key_fingerprint,
    list_ssh_keys,
    load_public_key,
    start_ssh_agent,
    test_connections,
)


GITHUB_SSH_URL = "https://github.com/settings/ssh/new"
BITBUCKET_SSH_URL = "https://bitbucket.org/account/settings/ssh-keys/"
STATE_FILE = Path.home() / ".ssh-key-gui-state.json"


LIGHT_STYLESHEET = """
QWidget { font-family: Segoe UI, Arial; font-size: 10pt; }
QMainWindow { background-color: #f0f3f8; }

QLabel#Title { font-size: 19pt; font-weight: 600; color: #152033; }
QLabel#Subtitle { color: #4e5d78; }
QLabel#StepDone { color: #1a7f37; font-weight: 600; }
QLabel#StepTodo { color: #4e5d78; }

QGroupBox {
    color: #152033;
    border: 1px solid #d5dceb;
    border-radius: 8px;
    margin-top: 10px;
    padding: 10px;
    background: #ffffff;
}
QGroupBox::title {
    subcontrol-origin: margin;
    left: 10px;
    padding: 0 6px;
    color: #4e5d78;
}

QLineEdit, QPlainTextEdit, QListWidget, QComboBox {
    background: #f7f9fd;
    border: 1px solid #d5dceb;
    border-radius: 6px;
    padding: 8px;
    color: #152033;
    selection-background-color: #1967d2;
}

QPlainTextEdit { font-family: Consolas, ui-monospace, monospace; font-size: 9.5pt; }

QPushButton {
    background: #f7f9fd;
    border: 1px solid #d5dceb;
    border-radius: 6px;
    padding: 8px 12px;
    color: #152033;
}
QPushButton:hover { background: #eef3fb; border-color: #bcc9e0; }
QPushButton:disabled { color: #8b96ac; background: #f7f9fd; border-color: #e2e8f3; }

QPushButton#Primary {
    background: #1967d2;
    border-color: #1967d2;
    color: #ffffff;
}
QPushButton#Primary:hover { background: #135abf; }

QPushButton#Danger {
    background: #cf3a2b;
    border-color: #cf3a2b;
    color: #ffffff;
}
QPushButton#Danger:hover { background: #bd3224; }

QFrame#BusyOverlay {
    background: rgba(240, 243, 248, 228);
    border-radius: 14px;
}

QPushButton#Cancel {
    background: #f7f9fd;
    border-color: #d5dceb;
    color: #cf3a2b;
}
"""


def _ts() -> str:
    return datetime.now().strftime("%H:%M:%S")


def _valid_key_name(name: str) -> bool:
    return bool(re.fullmatch(r"[A-Za-z0-9._-]+", name))


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
            if self._cancelled:
                return
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
        self.setWindowTitle("SSH Key Manager")
        self.resize(1080, 720)

        self._pool = QThreadPool.globalInstance()
        self._busy = False
        self._current_worker: _Worker | None = None

        self._selected_key: str | None = None
        self._used_keys: set[str] = set()
        self._copied_keys: set[str] = set()
        self._tested_keys_ok: dict[str, bool] = {}
        self._agent_loaded_keys: set[str] = set()
        self._state_loaded = False

        self._build_ui()
        self._wire_signals()
        self._load_state()
        self._refresh_keys(select_name=None)

    def _build_ui(self):  # pylint: disable=too-many-statements
        qt_app = cast(QApplication, QApplication.instance())
        qt_app.setStyleSheet(LIGHT_STYLESHEET)

        root = QWidget(self)
        self.setCentralWidget(root)
        outer = QVBoxLayout(root)
        outer.setContentsMargins(18, 16, 18, 12)
        outer.setSpacing(10)
        self._root = root

        title = QLabel("SSH Key Manager")
        title.setObjectName("Title")
        subtitle = QLabel("Minimal flow: generate key, add public key to host, verify connection.")
        subtitle.setObjectName("Subtitle")
        outer.addWidget(title)
        outer.addWidget(subtitle)

        splitter = QSplitter(Qt.Orientation.Horizontal)
        outer.addWidget(splitter, 1)

        sidebar = QWidget()
        sidebar_layout = QVBoxLayout(sidebar)
        sidebar_layout.setContentsMargins(0, 0, 0, 0)
        sidebar_layout.setSpacing(10)

        list_box = QGroupBox("Saved Keys")
        list_box_layout = QVBoxLayout(list_box)
        self.key_list = QListWidget()
        self.key_list.setAlternatingRowColors(True)
        list_box_layout.addWidget(self.key_list)
        sidebar_layout.addWidget(list_box, 1)

        side_actions = QHBoxLayout()
        self.btn_new_key = QPushButton("Generate")
        self.btn_new_key.setObjectName("Primary")
        self.btn_refresh_keys = QPushButton("Refresh")
        side_actions.addWidget(self.btn_new_key)
        side_actions.addWidget(self.btn_refresh_keys)
        sidebar_layout.addLayout(side_actions)

        self.content_stack = QStackedWidget()
        self.page_welcome = self._build_welcome_page()
        self.page_details = self._build_key_details_page()
        self.content_stack.addWidget(self.page_welcome)
        self.content_stack.addWidget(self.page_details)

        splitter.addWidget(sidebar)
        splitter.addWidget(self.content_stack)
        splitter.setSizes([320, 760])
        splitter.setStretchFactor(1, 1)

        activity_box = QGroupBox("Activity")
        activity_layout = QVBoxLayout(activity_box)
        self.activity_log = QPlainTextEdit()
        self.activity_log.setReadOnly(True)
        self.activity_log.setMaximumBlockCount(300)
        self.activity_log.setFixedHeight(120)
        activity_layout.addWidget(self.activity_log)
        outer.addWidget(activity_box)

        self._busy_overlay = QFrame(root)
        self._busy_overlay.setObjectName("BusyOverlay")
        self._busy_overlay.setVisible(False)
        self._busy_overlay.setFrameShape(QFrame.Shape.NoFrame)
        ov = QVBoxLayout(self._busy_overlay)
        ov.setContentsMargins(18, 18, 18, 18)
        ov.setSpacing(10)
        ov.addStretch(1)

        self._busy_label = QLabel("Working...")
        self._busy_label.setAlignment(Qt.AlignmentFlag.AlignHCenter)
        ov.addWidget(self._busy_label)

        self._busy_bar = QProgressBar()
        self._busy_bar.setRange(0, 0)
        self._busy_bar.setTextVisible(False)
        ov.addWidget(self._busy_bar)

        self._busy_cancel = QPushButton("Cancel")
        self._busy_cancel.setObjectName("Cancel")
        ov.addWidget(self._busy_cancel)
        ov.addStretch(2)

        self.setStatusBar(self.statusBar())
        self.statusBar().showMessage("Ready")
        self._update_overlay_geometry()

    def _build_welcome_page(self) -> QWidget:
        page = QWidget()
        layout = QVBoxLayout(page)
        layout.setContentsMargins(8, 4, 8, 4)
        layout.setSpacing(12)

        intro_box = QGroupBox("Quick Setup")
        intro_layout = QVBoxLayout(intro_box)
        intro = QLabel(
            "1) Start SSH agent. 2) Generate key. 3) Add key to agent and copy public key. "
            "4) Add to Git host. 5) Run connection test."
        )
        intro.setWordWrap(True)
        intro_layout.addWidget(intro)

        form = QFormLayout()
        self.input_key_name = QLineEdit("id_ed25519")
        self.input_key_name.setPlaceholderText("Example: id_work_github")

        self.combo_algorithm = QComboBox()
        self.combo_algorithm.addItems(["ed25519", "rsa", "ecdsa"])

        self.input_comment = QLineEdit()
        self.input_comment.setPlaceholderText("Optional comment (e.g. you@laptop)")

        self.input_passphrase = QLineEdit()
        self.input_passphrase.setPlaceholderText("Optional passphrase")
        self.input_passphrase.setEchoMode(QLineEdit.EchoMode.Password)

        self.input_passphrase_confirm = QLineEdit()
        self.input_passphrase_confirm.setPlaceholderText("Confirm passphrase")
        self.input_passphrase_confirm.setEchoMode(QLineEdit.EchoMode.Password)

        self.chk_show_passphrase = QCheckBox("Show passphrase")

        self.chk_force = QCheckBox("Overwrite if key already exists")
        self.chk_force.setChecked(False)

        form.addRow("Key name", self.input_key_name)
        form.addRow("Algorithm", self.combo_algorithm)
        form.addRow("Comment", self.input_comment)
        form.addRow("Passphrase", self.input_passphrase)
        form.addRow("Confirm", self.input_passphrase_confirm)
        form.addRow("", self.chk_show_passphrase)
        form.addRow("", self.chk_force)
        intro_layout.addLayout(form)

        self.btn_generate_welcome = QPushButton("Generate Key")
        self.btn_generate_welcome.setObjectName("Primary")
        self.btn_start_agent_welcome = QPushButton("Start / Check SSH Agent")
        self.btn_add_agent_welcome = QPushButton("Add Selected Key to Agent")
        intro_layout.addWidget(self.btn_start_agent_welcome)
        intro_layout.addWidget(self.btn_add_agent_welcome)
        intro_layout.addWidget(self.btn_generate_welcome)
        layout.addWidget(intro_box)

        guide_box = QGroupBox("Need Help?")
        guide_layout = QVBoxLayout(guide_box)
        guide = QLabel(
            "After generation, select the key from the left list, copy the public key, "
            "add it to your Git host, then run the connection test."
        )
        guide.setWordWrap(True)
        guide_layout.addWidget(guide)

        links = QHBoxLayout()
        self.btn_open_github_welcome = QPushButton("Open GitHub SSH Page")
        self.btn_open_bitbucket_welcome = QPushButton("Open Bitbucket SSH Page")
        links.addWidget(self.btn_open_github_welcome)
        links.addWidget(self.btn_open_bitbucket_welcome)
        guide_layout.addLayout(links)
        layout.addWidget(guide_box)

        layout.addStretch(1)
        return page

    def _build_key_details_page(self) -> QWidget:
        page = QWidget()
        layout = QVBoxLayout(page)
        layout.setContentsMargins(8, 4, 8, 4)
        layout.setSpacing(10)

        header_box = QGroupBox("Selected Key")
        header_layout = QVBoxLayout(header_box)
        self.key_details_name = QLabel("No key selected")
        self.key_details_name.setStyleSheet("font-size: 14pt; font-weight: 600;")
        self.key_details_path = QLabel("Path: -")
        self.key_details_path.setObjectName("Subtitle")
        self.key_details_fingerprint = QLabel("Fingerprint: -")
        self.key_details_fingerprint.setObjectName("Subtitle")
        header_layout.addWidget(self.key_details_name)
        header_layout.addWidget(self.key_details_path)
        header_layout.addWidget(self.key_details_fingerprint)
        layout.addWidget(header_box)

        checklist_box = QGroupBox("Guided Steps")
        checklist_layout = QVBoxLayout(checklist_box)
        self.step_generate = QLabel()
        self.step_agent = QLabel()
        self.step_copy = QLabel()
        self.step_test = QLabel()
        checklist_layout.addWidget(self.step_generate)
        checklist_layout.addWidget(self.step_agent)
        checklist_layout.addWidget(self.step_copy)
        checklist_layout.addWidget(self.step_test)
        layout.addWidget(checklist_box)

        key_box = QGroupBox("Public Key")
        key_layout = QVBoxLayout(key_box)
        self.public_key_text = QPlainTextEdit()
        self.public_key_text.setReadOnly(True)
        key_layout.addWidget(self.public_key_text)
        layout.addWidget(key_box, 1)

        action_row = QHBoxLayout()
        self.btn_copy_key = QPushButton("Copy Public Key")
        self.btn_copy_key.setObjectName("Primary")
        self.btn_add_key_agent = QPushButton("Add Key to SSH Agent")
        self.btn_mark_used = QPushButton("Mark as Added to Host")
        self.btn_delete_key = QPushButton("Delete Key")
        self.btn_delete_key.setObjectName("Danger")
        action_row.addWidget(self.btn_add_key_agent)
        action_row.addWidget(self.btn_copy_key)
        action_row.addWidget(self.btn_mark_used)
        action_row.addStretch(1)
        action_row.addWidget(self.btn_delete_key)
        layout.addLayout(action_row)

        self.btn_start_agent = QPushButton("Start / Check SSH Agent")
        layout.addWidget(self.btn_start_agent)

        host_row = QHBoxLayout()
        self.btn_open_github = QPushButton("Open GitHub SSH Settings")
        self.btn_open_bitbucket = QPushButton("Open Bitbucket SSH Settings")
        host_row.addWidget(self.btn_open_github)
        host_row.addWidget(self.btn_open_bitbucket)
        layout.addLayout(host_row)

        test_box = QGroupBox("Connection Test")
        test_layout = QVBoxLayout(test_box)
        self.test_status_label = QLabel("Not tested yet.")
        self.test_status_label.setObjectName("Subtitle")
        self.btn_run_test = QPushButton("Run SSH Test")
        self.test_result_text = QPlainTextEdit()
        self.test_result_text.setReadOnly(True)
        self.test_result_text.setFixedHeight(110)
        test_layout.addWidget(self.test_status_label)
        test_layout.addWidget(self.btn_run_test)
        test_layout.addWidget(self.test_result_text)
        layout.addWidget(test_box)

        return page

    def _wire_signals(self):
        self.btn_new_key.clicked.connect(self._on_generate_requested)
        self.btn_generate_welcome.clicked.connect(self._on_generate_requested)
        self.btn_start_agent_welcome.clicked.connect(self._start_agent)
        self.btn_add_agent_welcome.clicked.connect(self._add_selected_key_to_agent)
        self.btn_refresh_keys.clicked.connect(lambda: self._refresh_keys(self._selected_key))
        self.chk_show_passphrase.stateChanged.connect(self._toggle_passphrase_visibility)

        self.key_list.currentItemChanged.connect(self._on_selected_item_changed)

        self.btn_copy_key.clicked.connect(self._copy_selected_public_key)
        self.btn_add_key_agent.clicked.connect(self._add_selected_key_to_agent)
        self.btn_mark_used.clicked.connect(self._toggle_mark_used)
        self.btn_delete_key.clicked.connect(self._delete_selected_key)
        self.btn_run_test.clicked.connect(self._run_connection_test)
        self.btn_start_agent.clicked.connect(self._start_agent)

        self.btn_open_github.clicked.connect(lambda: self._open_host_page("github"))
        self.btn_open_bitbucket.clicked.connect(lambda: self._open_host_page("bitbucket"))
        self.btn_open_github_welcome.clicked.connect(lambda: self._open_host_page("github"))
        self.btn_open_bitbucket_welcome.clicked.connect(lambda: self._open_host_page("bitbucket"))

        self._busy_cancel.clicked.connect(self._cancel_operation)

    def resizeEvent(self, event):  # noqa: N802  # pylint: disable=invalid-name
        super().resizeEvent(event)
        self._update_overlay_geometry()

    def _update_overlay_geometry(self):
        self._busy_overlay.setGeometry(self._root.rect())

    def _load_state(self):
        if not STATE_FILE.exists():
            self._state_loaded = True
            return
        try:
            data = json.loads(STATE_FILE.read_text(encoding="utf-8"))
            self._used_keys = set(data.get("used_keys", []))
            self._copied_keys = set(data.get("copied_keys", []))
            tested_map = data.get("tested_keys_ok", {})
            if isinstance(tested_map, dict):
                self._tested_keys_ok = {str(k): bool(v) for k, v in tested_map.items()}
            self._agent_loaded_keys = set(data.get("agent_loaded_keys", []))
        except Exception as exc:  # noqa: BLE001  # pylint: disable=broad-exception-caught
            self._log(f"State load warning: {exc}")
        self._state_loaded = True

    def _save_state(self):
        if not self._state_loaded:
            return
        data = {
            "used_keys": sorted(self._used_keys),
            "copied_keys": sorted(self._copied_keys),
            "tested_keys_ok": self._tested_keys_ok,
            "agent_loaded_keys": sorted(self._agent_loaded_keys),
        }
        try:
            STATE_FILE.write_text(json.dumps(data, indent=2), encoding="utf-8")
        except Exception as exc:  # noqa: BLE001  # pylint: disable=broad-exception-caught
            self._log(f"State save warning: {exc}")

    def _toggle_passphrase_visibility(self):
        echo = QLineEdit.EchoMode.Normal if self.chk_show_passphrase.isChecked() else QLineEdit.EchoMode.Password
        self.input_passphrase.setEchoMode(echo)
        self.input_passphrase_confirm.setEchoMode(echo)

    def _log(self, message: str):
        self.activity_log.appendPlainText(f"[{_ts()}] {message}")

    def _set_busy(self, busy: bool, message: str | None = None):
        self._busy = busy
        for btn in (
            self.btn_new_key,
            self.btn_generate_welcome,
            self.btn_start_agent_welcome,
            self.btn_add_agent_welcome,
            self.btn_refresh_keys,
            self.btn_add_key_agent,
            self.btn_copy_key,
            self.btn_mark_used,
            self.btn_delete_key,
            self.btn_run_test,
            self.btn_start_agent,
            self.btn_open_github,
            self.btn_open_bitbucket,
            self.btn_open_github_welcome,
            self.btn_open_bitbucket_welcome,
        ):
            btn.setEnabled(not busy)

        if message:
            self.statusBar().showMessage(message)

        if busy:
            self._busy_label.setText(message or "Working...")
            self._busy_overlay.setVisible(True)
            self._busy_overlay.raise_()
        else:
            self._busy_overlay.setVisible(False)
            self._current_worker = None

    def _cancel_operation(self):
        if self._current_worker:
            self._current_worker.cancel()
            self._current_worker = None
            self._set_busy(False, "Operation cancelled")
            self._log("Cancelled current operation")

    def _run_worker(
        self,
        fn,
        on_done,
        busy_message: str,
        error_title: str,
    ):
        if self._busy:
            return
        self._set_busy(True, busy_message)
        worker = _Worker(fn)
        self._current_worker = worker

        def done(result: Any):
            self._set_busy(False)
            on_done(result)

        def failed(err: str):
            self._set_busy(False)
            self._log(f"Error: {err}")
            QMessageBox.critical(self, error_title, err)

        worker.signals.finished.connect(done)
        worker.signals.failed.connect(failed)
        self._pool.start(worker)

    def _refresh_keys(self, select_name: str | None):
        keys = sorted(list_ssh_keys(), key=lambda it: it["name"].lower())
        self.key_list.clear()

        for key in keys:
            name = key["name"]
            tags = []
            if name in self._used_keys:
                tags.append("added")
            if self._tested_keys_ok.get(name):
                tags.append("tested")
            if name in self._agent_loaded_keys:
                tags.append("agent")
            suffix = f"  [{' | '.join(tags)}]" if tags else ""
            item = QListWidgetItem(f"{name}{suffix}")
            item.setData(Qt.ItemDataRole.UserRole, key)
            self.key_list.addItem(item)

        if not keys:
            self._selected_key = None
            self.content_stack.setCurrentWidget(self.page_welcome)
            self.statusBar().showMessage("No SSH keys found. Generate one to get started.")
            self._update_guided_steps()
            return

        target = select_name or self._selected_key or keys[0]["name"]
        for index in range(self.key_list.count()):
            item = self.key_list.item(index)
            data = cast(dict[str, str], item.data(Qt.ItemDataRole.UserRole))
            if data["name"] == target:
                self.key_list.setCurrentItem(item)
                return

        self.key_list.setCurrentRow(0)

    def _on_selected_item_changed(self, current: QListWidgetItem | None, _previous: QListWidgetItem | None):
        if current is None:
            self._selected_key = None
            self.content_stack.setCurrentWidget(self.page_welcome)
            self._update_guided_steps()
            return

        key_info = cast(dict[str, str], current.data(Qt.ItemDataRole.UserRole))
        key_name = key_info["name"]
        key_path = key_info["path"]

        self._selected_key = key_name
        self.content_stack.setCurrentWidget(self.page_details)

        public_key = load_public_key(key_name) or ""
        fingerprint = get_key_fingerprint(key_name) or "Unknown"
        self.key_details_name.setText(key_name)
        self.key_details_path.setText(f"Path: {key_path}")
        self.key_details_fingerprint.setText(f"Fingerprint: {fingerprint}")
        self.public_key_text.setPlainText(public_key)
        self.test_result_text.clear()

        if self._tested_keys_ok.get(key_name):
            self.test_status_label.setText("Last test: OK")
        else:
            self.test_status_label.setText("Last test: not successful yet")

        self._update_mark_used_button()
        self._update_guided_steps()

    def _on_generate_requested(self):
        if self._busy:
            return

        key_name = self.input_key_name.text().strip()
        if not key_name:
            QMessageBox.warning(self, "Invalid key name", "Key name cannot be empty.")
            return
        if not _valid_key_name(key_name):
            QMessageBox.warning(
                self,
                "Invalid key name",
                "Use only letters, numbers, dot (.), underscore (_) or dash (-).",
            )
            return

        algorithm = cast(KeyAlgorithm, self.combo_algorithm.currentText())
        comment = self.input_comment.text().strip() or None
        passphrase_text = self.input_passphrase.text()
        passphrase_confirm = self.input_passphrase_confirm.text()
        if passphrase_text != passphrase_confirm:
            QMessageBox.warning(self, "Passphrase mismatch", "Passphrase and confirmation do not match.")
            return
        passphrase = passphrase_text if passphrase_text else None
        force = self.chk_force.isChecked()

        def generate_task():
            return generate_key(
                algorithm=algorithm,
                key_name=key_name,
                comment=comment,
                passphrase=passphrase,
                force=force,
            )

        def after_generate(message: str):
            ok = "generated successfully" in message.lower()
            self._log(message)
            self.statusBar().showMessage(message)
            if ok:
                self.input_passphrase.clear()
                self.input_passphrase_confirm.clear()
                self._refresh_keys(select_name=key_name)
            else:
                QMessageBox.warning(self, "Key Generation", message)

        self._run_worker(
            fn=generate_task,
            on_done=after_generate,
            busy_message="Generating SSH key...",
            error_title="Key generation failed",
        )

    def _copy_selected_public_key(self):
        if not self._selected_key:
            QMessageBox.information(self, "No key selected", "Select a key from the left list first.")
            return
        pub = load_public_key(self._selected_key)
        if not pub:
            QMessageBox.warning(self, "Public key missing", "Could not load the selected public key.")
            return

        clipboard = QGuiApplication.clipboard()
        clipboard.setText(pub)
        self._copied_keys.add(self._selected_key)
        self._save_state()
        self._log(f"Copied public key for '{self._selected_key}' to clipboard")
        self.statusBar().showMessage("Public key copied to clipboard")
        self._update_guided_steps()

    def _toggle_mark_used(self):
        if not self._selected_key:
            QMessageBox.information(self, "No key selected", "Select a key from the left list first.")
            return

        key = self._selected_key
        if key in self._used_keys:
            self._used_keys.remove(key)
            self._log(f"Unmarked '{key}' as added to host")
        else:
            self._used_keys.add(key)
            self._log(f"Marked '{key}' as added to host")

        self._update_mark_used_button()
        self._refresh_keys(select_name=key)
        self._update_guided_steps()
        self._save_state()

    def _update_mark_used_button(self):
        if self._selected_key and self._selected_key in self._used_keys:
            self.btn_mark_used.setText("Unmark as Added")
        else:
            self.btn_mark_used.setText("Mark as Added to Host")

    def _start_agent(self):
        if self._busy:
            return

        def done(message: str):
            self._log(message)
            self.statusBar().showMessage(message)
            if "could not" in message.lower() or "not reachable" in message.lower():
                QMessageBox.warning(self, "SSH Agent", message)

        self._run_worker(
            fn=start_ssh_agent,
            on_done=done,
            busy_message="Starting/checking SSH agent...",
            error_title="SSH agent check failed",
        )

    def _add_selected_key_to_agent(self):
        if self._busy:
            return
        if not self._selected_key:
            QMessageBox.information(self, "No key selected", "Select a key from the left list first.")
            return

        key_name = self._selected_key

        def done(message: str):
            self._log(message)
            self.statusBar().showMessage(message)
            if "added" in message.lower() and "failed" not in message.lower():
                self._agent_loaded_keys.add(key_name)
                self._save_state()
                self._refresh_keys(select_name=key_name)
                self._update_guided_steps()
            else:
                QMessageBox.warning(self, "Add key to agent", message)

        self._run_worker(
            fn=lambda: add_key_to_agent(key_name),
            on_done=done,
            busy_message="Adding selected key to SSH agent...",
            error_title="Failed to add key to agent",
        )

    def _delete_selected_key(self):
        if not self._selected_key:
            QMessageBox.information(self, "No key selected", "Select a key from the left list first.")
            return

        key_name = self._selected_key
        confirm = QMessageBox.question(
            self,
            "Delete key",
            f"Delete '{key_name}' private and public key files from ~/.ssh?",
            QMessageBox.StandardButton.Yes | QMessageBox.StandardButton.No,
            QMessageBox.StandardButton.No,
        )
        if confirm != QMessageBox.StandardButton.Yes:
            return

        ssh_dir = Path.home() / ".ssh"
        private_path = ssh_dir / key_name
        public_path = ssh_dir / f"{key_name}.pub"

        private_path.unlink(missing_ok=True)
        public_path.unlink(missing_ok=True)

        self._used_keys.discard(key_name)
        self._copied_keys.discard(key_name)
        self._tested_keys_ok.pop(key_name, None)
        self._agent_loaded_keys.discard(key_name)
        self._save_state()

        self._log(f"Deleted key '{key_name}'")
        self.statusBar().showMessage(f"Deleted '{key_name}'")
        self._refresh_keys(select_name=None)

    def _open_host_page(self, host: str):
        url = GITHUB_SSH_URL if host == "github" else BITBUCKET_SSH_URL
        webbrowser.open(url)
        self._log(f"Opened {host} SSH settings page")

    def _run_connection_test(self):
        if not self._selected_key:
            QMessageBox.information(self, "No key selected", "Select a key from the left list first.")
            return

        key_name = self._selected_key
        self.test_result_text.clear()
        self.test_status_label.setText("Testing SSH authentication...")

        def after_test(results: dict[str, tuple[bool, str]]):
            self._handle_test_results(key_name, results)

        self._run_worker(
            fn=lambda: test_connections(key_name=key_name),
            on_done=after_test,
            busy_message="Running SSH connection tests...",
            error_title="Connection test failed",
        )

    def _handle_test_results(self, key_name: str, results: dict[str, tuple[bool, str]]):
        lines: list[str] = []
        any_ok = False

        for host, (ok, msg) in results.items():
            status = "OK" if ok else "FAIL"
            lines.append(f"{host}: {status}")
            lines.append(msg)
            lines.append("")
            any_ok = any_ok or ok

        self.test_result_text.setPlainText("\n".join(lines).strip())
        self._tested_keys_ok[key_name] = any_ok
        self._save_state()

        if any_ok:
            self.test_status_label.setText("Last test: success")
            self.statusBar().showMessage("SSH test passed for at least one host")
            self._log(f"SSH test successful for '{key_name}'")
        else:
            self.test_status_label.setText("Last test: failed")
            self.statusBar().showMessage("SSH test failed")
            self._log(f"SSH test failed for '{key_name}'")
            QMessageBox.warning(
                self,
                "SSH test failed",
                "Authentication did not succeed. Ensure your public key is added to your Git host.",
            )

        self._refresh_keys(select_name=key_name)
        self._update_guided_steps()

    def _step_line(self, done: bool, text: str) -> str:
        prefix = "[Done]" if done else "[Todo]"
        return f"{prefix} {text}"

    def _update_guided_steps(self):
        has_key = bool(self._selected_key)
        in_agent = bool(self._selected_key and self._selected_key in self._agent_loaded_keys)
        copied = bool(self._selected_key and self._selected_key in self._copied_keys)
        tested = bool(self._selected_key and self._tested_keys_ok.get(self._selected_key))

        self.step_generate.setText(self._step_line(has_key, "Select or generate a key"))
        self.step_agent.setText(self._step_line(in_agent, "Start agent and add selected key"))
        self.step_copy.setText(self._step_line(copied, "Copy public key and add it to Git host"))
        self.step_test.setText(self._step_line(tested, "Run SSH test to verify authentication"))

        self.step_generate.setObjectName("StepDone" if has_key else "StepTodo")
        self.step_agent.setObjectName("StepDone" if in_agent else "StepTodo")
        self.step_copy.setObjectName("StepDone" if copied else "StepTodo")
        self.step_test.setObjectName("StepDone" if tested else "StepTodo")

        self.step_generate.style().polish(self.step_generate)
        self.step_agent.style().polish(self.step_agent)
        self.step_copy.style().polish(self.step_copy)
        self.step_test.style().polish(self.step_test)


if __name__ == "__main__":
    app = QApplication(sys.argv)
    win = SSHApp()
    win.show()
    sys.exit(app.exec())
