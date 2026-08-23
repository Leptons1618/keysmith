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
    QSizePolicy,
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


STYLESHEET = """
QWidget { font-family: 'Segoe UI', 'Ubuntu', 'Helvetica Neue', Arial, sans-serif; font-size: 10pt; color: #1b2025; }
QMainWindow, QDialog { background: #eef0f3; }

QLabel#Title { font-size: 17pt; font-weight: 700; color: #1b2025; }
QLabel#Subtitle { color: #585e66; margin-top: 2px; }
QLabel#KeyName { font-size: 15pt; font-weight: 700; }
QLabel#MetaLine { color: #585e66; font-size: 9.5pt; }
QLabel#StepTodo { color: #585e66; }
QLabel#StepDone { color: #006d68; font-weight: 600; }
QLabel#WarningText { color: #895c07; font-weight: 600; }
QLabel#Badge {
    font-size: 8.5pt; font-weight: 700; color: #585e66;
    border: 1px solid #d5d8db; border-radius: 2px;
    padding: 2px 8px; background: transparent;
}
QLabel#BadgeOn { color: #006d68; border-color: #72aba7; background: #d2efec; }
QLabel#BadgeWarn { color: #895c07; border-color: #cfa954; background: #feecd3; }
QLabel#EmptyHint { color: #585e66; }

QGroupBox {
    font-weight: 600;
    color: #1b2025;
    border: 1px solid #d5d8db;
    border-radius: 2px;
    background: #f5f7f9;
    padding: 8px;
    margin-top: 0;
}
QGroupBox::title {
    subcontrol-origin: margin;
    left: 8px;
    padding: 0 4px;
    background: #eef0f3;
    color: #585e66;
    font-size: 8.5pt;
    font-weight: 700;
    letter-spacing: 1px;
}

QLineEdit, QPlainTextEdit, QComboBox, QListWidget {
    background: #fbfcfd;
    border: 1px solid #d5d8db;
    border-radius: 2px;
    padding: 6px 8px;
    color: #1b2025;
    selection-background-color: #1b2025;
    selection-color: #ffffff;
}
QLineEdit:focus, QPlainTextEdit:focus, QComboBox:focus, QListWidget:focus { border-color: #006d68; }
QLineEdit:disabled { color: #71757a; background: #eef0f3; }
QPlainTextEdit { font-family: 'Consolas', 'DejaVu Sans Mono', ui-monospace, monospace; font-size: 9pt; }
QListWidget { padding: 2px; background: #fbfcfd; }
QListWidget::item { padding: 7px 8px; border-bottom: 1px solid #e2e5ea; }
QListWidget::item:selected { background: #1b2025; color: #ffffff; }
QListWidget::item:hover:!selected { background: #dadee3; }

QComboBox::drop-down { border: none; width: 20px; }
QComboBox::down-arrow {
    width: 0; height: 0;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-top: 5px solid #1b2025;
    margin-right: 6px;
}
QComboBox QAbstractItemView {
    background: #fbfcfd;
    border: 1px solid #d5d8db;
    selection-background-color: #006d68;
    selection-color: #ffffff;
}

QCheckBox { spacing: 8px; }
QCheckBox::indicator {
    width: 15px; height: 15px;
    border: 1px solid #b4b8bc;
    border-radius: 2px;
    background: #fbfcfd;
}
QCheckBox::indicator:hover { border-color: #006d68; }
QCheckBox::indicator:checked { background: #006d68; border-color: #006d68; }

QPushButton {
    background: transparent;
    border: 1px solid #1b2025;
    border-radius: 2px;
    padding: 7px 16px;
    color: #1b2025;
    font-weight: 600;
}
QPushButton:hover { background: #006d68; color: #ffffff; }
QPushButton:pressed { background: #31363c; color: #ffffff; }
QPushButton:focus { background: #dadee3; color: #1b2025; }
QPushButton:disabled { color: #71757a; border-color: #c3c8cd; background: transparent; }

QPushButton#Primary { background: #006d68; color: #ffffff; }
QPushButton#Primary:hover { background: #005a55; }
QPushButton#Primary:pressed { background: #004a46; }
QPushButton#Primary:focus { background: #005a55; border: 1px dashed #ffffff; color: #ffffff; }
QPushButton#Primary:disabled { background: #c3c8cd; border-color: #c3c8cd; color: #eef0f3; }

QPushButton#Danger { background: transparent; border-color: #9e2d28; color: #9e2d28; }
QPushButton#Danger:hover { background: #9e2d28; color: #ffffff; }
QPushButton#Danger:pressed { background: #861213; color: #ffffff; }
QPushButton#Danger:focus { background: #fde7e4; color: #9e2d28; border: 1px dashed #9e2d28; }
QPushButton#Danger:disabled { color: #71757a; border-color: #c3c8cd; background: transparent; }

QPushButton#Ghost { border-color: #b4b8bc; font-weight: 600; }
QPushButton#Ghost:hover { background: #006d68; color: #ffffff; border-color: #006d68; }
QPushButton#Ghost:focus { background: #dadee3; }

QPushButton#Cancel { background: rgba(27, 32, 37, 205); border-color: #b4b8bc; color: #eef0f3; }
QPushButton#Cancel:hover { background: #eef0f3; color: #1b2025; }

QFrame#BusyOverlay { background: rgba(27, 32, 37, 205); }
QLabel#BusyText { color: #eef0f3; font-size: 11pt; font-weight: 600; }
QFrame#BusyOverlay QProgressBar { background: #31363c; border: none; }
QFrame#BusyOverlay QProgressBar::chunk { background: #eef0f3; }

QProgressBar { background: #d5d8db; border: none; min-height: 4px; max-height: 4px; }
QProgressBar::chunk { background: #006d68; }

QScrollBar:vertical { background: transparent; width: 10px; margin: 0; }
QScrollBar::handle:vertical { background: #c3c8cd; min-height: 30px; border-radius: 0; }
QScrollBar::handle:vertical:hover { background: #71757a; }
QScrollBar:horizontal { background: transparent; height: 10px; margin: 0; }
QScrollBar::handle:horizontal { background: #c3c8cd; min-width: 30px; border-radius: 0; }
QScrollBar::handle:horizontal:hover { background: #71757a; }
QScrollBar::add-line, QScrollBar::sub-line { width: 0; height: 0; }
QScrollBar::add-page, QScrollBar::sub-page { background: transparent; }

QSplitter::handle { background: #eef0f3; width: 1px; }
QSplitter::handle:hover { background: #006d68; }

QStatusBar { background: #eef0f3; color: #585e66; border-top: 1px solid #d5d8db; }
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
        qt_app.setStyleSheet(STYLESHEET)

        self._root = QWidget(self)
        self.setCentralWidget(self._root)
        outer = QVBoxLayout(self._root)
        outer.setContentsMargins(14, 12, 14, 6)
        outer.setSpacing(8)

        # --- header -------------------------------------------------------
        header_row = QHBoxLayout()
        header_row.setSpacing(16)
        heading = QVBoxLayout()
        heading.setSpacing(0)
        title = QLabel("SSH Key Manager")
        title.setObjectName("Title")
        subtitle = QLabel("Generate a key, hand the public half to your Git host, verify the connection.")
        subtitle.setObjectName("Subtitle")
        heading.addWidget(title)
        heading.addWidget(subtitle)
        header_row.addLayout(heading)
        header_row.addStretch(1)

        agent_col = QVBoxLayout()
        agent_col.setSpacing(3)
        self.lbl_agent_status = self._make_badge("AGENT")
        btn_header_agent = QPushButton("Check agent")
        btn_header_agent.setObjectName("Ghost")
        btn_header_agent.clicked.connect(self._start_agent)
        agent_col.addWidget(self.lbl_agent_status, 0, Qt.AlignmentFlag.AlignRight)
        agent_col.addWidget(btn_header_agent, 0, Qt.AlignmentFlag.AlignRight)
        header_row.addLayout(agent_col)
        self.btn_start_agent = btn_header_agent

        outer.addLayout(header_row)

        # --- main vertical split: workbench over activity strip ------------
        v_split = QSplitter(Qt.Orientation.Vertical)
        outer.addWidget(v_split, 1)

        # --- keys rail ------------------------------------------------------
        rail_box = QGroupBox("KEYS")
        rail_layout = QVBoxLayout(rail_box)
        rail_layout.setContentsMargins(10, 14, 10, 10)
        rail_layout.setSpacing(8)

        self.key_list = QListWidget()
        self.key_list.setAlternatingRowColors(False)
        self.key_list.setTextElideMode(Qt.TextElideMode.ElideMiddle)
        rail_layout.addWidget(self.key_list, 1)

        gen_form = QFormLayout()
        gen_form.setSpacing(6)
        self.input_key_name = QLineEdit("id_ed25519")
        self.input_key_name.setPlaceholderText("e.g. id_work_github")
        self.combo_algorithm = QComboBox()
        self.combo_algorithm.addItems(["ed25519", "rsa", "ecdsa"])
        self.input_comment = QLineEdit()
        self.input_comment.setPlaceholderText("you@laptop")
        gen_form.addRow("Name", self.input_key_name)
        gen_form.addRow("Type", self.combo_algorithm)
        gen_form.addRow("Comment", self.input_comment)
        rail_layout.addLayout(gen_form)

        pass_row = QHBoxLayout()
        pass_row.setSpacing(6)
        self.input_passphrase = QLineEdit()
        self.input_passphrase.setPlaceholderText("Passphrase")
        self.input_passphrase.setEchoMode(QLineEdit.EchoMode.Password)
        self.input_passphrase_confirm = QLineEdit()
        self.input_passphrase_confirm.setPlaceholderText("Confirm")
        self.input_passphrase_confirm.setEchoMode(QLineEdit.EchoMode.Password)
        pass_row.addWidget(self.input_passphrase)
        pass_row.addWidget(self.input_passphrase_confirm)
        rail_layout.addLayout(pass_row)

        opts_row = QHBoxLayout()
        opts_row.setSpacing(12)
        self.chk_show_passphrase = QCheckBox("Show")
        self.chk_force = QCheckBox("Overwrite existing")
        opts_row.addWidget(self.chk_show_passphrase)
        opts_row.addWidget(self.chk_force)
        opts_row.addStretch(1)
        rail_layout.addLayout(opts_row)

        self.btn_generate_welcome = QPushButton("Generate key")
        self.btn_generate_welcome.setObjectName("Primary")
        rail_layout.addWidget(self.btn_generate_welcome)

        rail_footer = QHBoxLayout()
        rail_footer.setSpacing(6)
        hint = QLabel("Passphrases stay on this machine.")
        hint.setObjectName("EmptyHint")
        rail_footer.addWidget(hint, 1)
        self.btn_refresh_keys = QPushButton("Refresh")
        rail_footer.addWidget(self.btn_refresh_keys)
        rail_layout.addLayout(rail_footer)

        # --- right pane stack ----------------------------------------------
        self.content_stack = QStackedWidget()
        self.page_welcome = self._build_welcome_page()
        self.page_details = self._build_key_details_page()
        self.content_stack.addWidget(self.page_welcome)
        self.content_stack.addWidget(self.page_details)

        h_split = QSplitter(Qt.Orientation.Horizontal)
        h_split.setChildrenCollapsible(False)
        h_split.addWidget(rail_box)
        h_split.addWidget(self.content_stack)
        h_split.setSizes([330, 850])
        h_split.setStretchFactor(0, 0)
        h_split.setStretchFactor(1, 1)
        v_split.addWidget(h_split)
        v_split.setChildrenCollapsible(False)

        # workbench floor: the details page's natural minimum, so the
        # activity strip can shrink but never crush it
        h_min = self.page_details.minimumSizeHint().height()
        v_split.widget(0).setMinimumHeight(h_min + 24)

        # --- activity strip --------------------------------------------------
        activity_box = QGroupBox("ACTIVITY")
        activity_layout = QVBoxLayout(activity_box)
        activity_layout.setContentsMargins(10, 14, 10, 8)
        self.activity_log = QPlainTextEdit()
        self.activity_log.setReadOnly(True)
        self.activity_log.setMaximumBlockCount(300)
        self.activity_log.setMinimumHeight(40)
        activity_layout.addWidget(self.activity_log)
        v_split.addWidget(activity_box)
        v_split.setSizes([760, 130])
        v_split.setStretchFactor(0, 1)
        self._v_split = v_split
        self._h_split = h_split

        # --- busy overlay ----------------------------------------------------
        self._busy_overlay = QFrame(self._root)
        self._busy_overlay.setObjectName("BusyOverlay")
        self._busy_overlay.setVisible(False)
        self._busy_overlay.setFrameShape(QFrame.Shape.NoFrame)
        self._busy_overlay.setAttribute(Qt.WidgetAttribute.WA_StyledBackground, True)
        ov = QVBoxLayout(self._busy_overlay)
        ov.setContentsMargins(18, 18, 18, 18)
        ov.setSpacing(12)
        ov.addStretch(1)

        self._busy_label = QLabel("Working...")
        self._busy_label.setObjectName("BusyText")
        self._busy_label.setAlignment(Qt.AlignmentFlag.AlignHCenter)
        ov.addWidget(self._busy_label)

        self._busy_bar = QProgressBar()
        self._busy_bar.setRange(0, 0)
        self._busy_bar.setTextVisible(False)
        ov.addWidget(self._busy_bar)

        self._busy_cancel = QPushButton("Cancel operation")
        self._busy_cancel.setObjectName("Cancel")
        ov.addWidget(self._busy_cancel, 0, Qt.AlignmentFlag.AlignHCenter)
        ov.addStretch(2)

        self.setStatusBar(self.statusBar())
        self.statusBar().showMessage("Ready")
        self._update_overlay_geometry()

    def _make_badge(self, text: str) -> QLabel:
        badge = QLabel(text)
        badge.setObjectName("Badge")
        return badge

    def _elide(self, label: QLabel, text: str) -> str:
        metrics = label.fontMetrics()
        return metrics.elidedText(text, Qt.TextElideMode.ElideMiddle, max(40, label.width() - 4))

    def _set_key_name_display(self, key_name: str):
        label = self.key_details_name
        metrics = label.fontMetrics()
        elided = metrics.elidedText(
            key_name, Qt.TextElideMode.ElideMiddle, max(60, label.width() - 8)
        )
        label.setText(elided)

    def _build_welcome_page(self) -> QWidget:
        page = QWidget()
        layout = QVBoxLayout(page)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(10)

        intro_box = QGroupBox("SETUP PATH")
        intro_layout = QVBoxLayout(intro_box)
        intro_layout.setSpacing(8)
        intro = QLabel(
            "1 · Start or check the SSH agent\n"
            "2 · Name the key on the left and press Generate key\n"
            "3 · Copy the public key and add it to your Git host\n"
            "4 · Load the private key into the agent\n"
            "5 · Run the connection test until both hosts answer"
        )
        intro.setWordWrap(True)
        intro_layout.addWidget(intro)

        empty_hint = QLabel("No key selected yet. Generate one on the left, or pick an existing key.")
        empty_hint.setObjectName("EmptyHint")
        intro_layout.addWidget(empty_hint)
        layout.addWidget(intro_box)

        guide_box = QGroupBox("GIT HOSTS")
        guide_layout = QVBoxLayout(guide_box)
        guide = QLabel("Open the SSH settings page, paste the public key, save.")
        guide.setObjectName("Subtitle")
        guide_layout.addWidget(guide)

        links = QHBoxLayout()
        links.setSpacing(8)
        self.btn_open_github_welcome = QPushButton("GitHub SSH settings")
        self.btn_open_bitbucket_welcome = QPushButton("Bitbucket SSH settings")
        links.addWidget(self.btn_open_github_welcome)
        links.addWidget(self.btn_open_bitbucket_welcome)
        links.addStretch(1)
        guide_layout.addLayout(links)
        layout.addWidget(guide_box)

        layout.addStretch(1)
        return page

    def _build_key_details_page(self) -> QWidget:
        page = QWidget()
        layout = QVBoxLayout(page)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(10)

        identity_box = QGroupBox("IDENTITY")
        identity_layout = QVBoxLayout(identity_box)
        identity_layout.setSpacing(6)
        name_row = QHBoxLayout()
        name_row.setSpacing(10)
        self.key_details_name = QLabel("No key selected")
        self.key_details_name.setObjectName("KeyName")
        self.key_details_name.setSizePolicy(QSizePolicy.Policy.Ignored, QSizePolicy.Policy.Preferred)
        self.key_details_name.setMinimumWidth(1)
        self.key_details_name.setMouseTracking(True)
        name_row.addWidget(self.key_details_name, 1)
        name_row.addSpacing(4)
        self.lbl_badge_agent = self._make_badge("IN AGENT")
        self.lbl_badge_used = self._make_badge("ADDED TO HOST")
        self.lbl_badge_tested = self._make_badge("TESTED OK")
        name_row.addWidget(self.lbl_badge_agent)
        name_row.addWidget(self.lbl_badge_used)
        name_row.addWidget(self.lbl_badge_tested)
        name_row.addStretch(1)
        identity_layout.addLayout(name_row)

        self.key_details_fingerprint = QLabel()
        self.key_details_fingerprint.setObjectName("MetaLine")
        self.key_details_fingerprint.setWordWrap(True)
        self.key_details_path = QLabel()
        self.key_details_path.setObjectName("MetaLine")
        self.key_details_path.setWordWrap(True)
        identity_layout.addWidget(self.key_details_fingerprint)
        identity_layout.addWidget(self.key_details_path)
        layout.addWidget(identity_box)

        steps_box = QGroupBox("PROGRESS")
        checklist_layout = QVBoxLayout(steps_box)
        checklist_layout.setSpacing(4)
        self.step_generate = QLabel()
        self.step_agent = QLabel()
        self.step_copy = QLabel()
        self.step_test = QLabel()
        checklist_layout.addWidget(self.step_generate)
        checklist_layout.addWidget(self.step_agent)
        checklist_layout.addWidget(self.step_copy)
        checklist_layout.addWidget(self.step_test)
        layout.addWidget(steps_box)

        key_box = QGroupBox("PUBLIC KEY")
        key_layout = QVBoxLayout(key_box)
        self.public_key_text = QPlainTextEdit()
        self.public_key_text.setReadOnly(True)
        self.public_key_text.setMinimumHeight(44)
        key_layout.addWidget(self.public_key_text)
        layout.addWidget(key_box, 1)

        action_row = QHBoxLayout()
        action_row.setSpacing(8)
        self.btn_copy_key = QPushButton("Copy public key")
        self.btn_copy_key.setObjectName("Primary")
        self.btn_add_key_agent = QPushButton("Add to agent")
        self.btn_mark_used = QPushButton("Mark as added to host")
        self.btn_delete_key = QPushButton("Delete key")
        self.btn_delete_key.setObjectName("Danger")
        action_row.addWidget(self.btn_copy_key)
        action_row.addWidget(self.btn_add_key_agent)
        action_row.addWidget(self.btn_mark_used)
        action_row.addStretch(1)
        action_row.addWidget(self.btn_delete_key)
        layout.addLayout(action_row)

        test_box = QGroupBox("CONNECTION TEST")
        test_layout = QVBoxLayout(test_box)
        test_layout.setSpacing(8)
        status_row = QHBoxLayout()
        status_row.setSpacing(10)
        self.test_status_label = QLabel("Not tested yet.")
        self.test_status_label.setObjectName("Subtitle")
        status_row.addWidget(self.test_status_label, 1)
        self.btn_run_test = QPushButton("Run SSH test")
        status_row.addWidget(self.btn_run_test)
        test_layout.addLayout(status_row)
        self.test_result_text = QPlainTextEdit()
        self.test_result_text.setReadOnly(True)
        self.test_result_text.setFixedHeight(48)
        test_layout.addWidget(self.test_result_text)
        layout.addWidget(test_box)

        return page

    def _wire_signals(self):
        self.btn_generate_welcome.clicked.connect(self._on_generate_requested)
        self.btn_refresh_keys.clicked.connect(lambda: self._refresh_keys(self._selected_key))
        self.chk_show_passphrase.stateChanged.connect(self._toggle_passphrase_visibility)

        self.key_list.currentItemChanged.connect(self._on_selected_item_changed)

        self.btn_copy_key.clicked.connect(self._copy_selected_public_key)
        self.btn_add_key_agent.clicked.connect(self._add_selected_key_to_agent)
        self.btn_mark_used.clicked.connect(self._toggle_mark_used)
        self.btn_delete_key.clicked.connect(self._delete_selected_key)
        self.btn_run_test.clicked.connect(self._run_connection_test)

        self.btn_open_github_welcome.clicked.connect(lambda: self._open_host_page("github"))
        self.btn_open_bitbucket_welcome.clicked.connect(lambda: self._open_host_page("bitbucket"))

        self._busy_cancel.clicked.connect(self._cancel_operation)

    def _set_badge(self, badge: QLabel, on: bool):
        badge.setObjectName("BadgeOn" if on else "Badge")
        badge.style().unpolish(badge)
        badge.style().polish(badge)

    def _set_agent_badge(self, state: str):
        text = {"unknown": "AGENT ?", "on": "AGENT ON", "off": "AGENT OFF"}.get(state, "AGENT ?")
        style = {"on": "BadgeOn", "off": "BadgeWarn"}.get(state, "Badge")
        badge = self.lbl_agent_status
        badge.setText(text)
        badge.setObjectName(style)
        badge.style().unpolish(badge)
        badge.style().polish(badge)

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
            self.btn_generate_welcome,
            self.btn_refresh_keys,
            self.btn_start_agent,
            self.btn_add_key_agent,
            self.btn_copy_key,
            self.btn_mark_used,
            self.btn_delete_key,
            self.btn_run_test,
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
            item = QListWidgetItem(key["name"])
            item.setData(Qt.ItemDataRole.UserRole, key)
            item.setToolTip(key["name"])
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

        self._selected_key = key_name
        self.content_stack.setCurrentWidget(self.page_details)

        self._set_key_name_display(key_name)
        public_key = load_public_key(key_name) or ""
        fingerprint = get_key_fingerprint(key_name) or "Unknown"
        self.key_details_name.setToolTip(key_name)
        self.key_details_path.setToolTip(str(Path.home() / ".ssh" / key_name))
        self.key_details_fingerprint.setText(
            f"Fingerprint {self._elide(self.key_details_fingerprint, fingerprint)}"
        )
        private_path = str(Path.home() / ".ssh" / key_name)
        self.key_details_path.setText(
            f"Private key {self._elide(self.key_details_path, private_path)}"
        )
        self.public_key_text.setPlainText(public_key)
        self.test_result_text.clear()

        if self._tested_keys_ok.get(key_name):
            self.test_status_label.setText("Last test: success")
        else:
            self.test_status_label.setText("Last test: failed or not run yet")

        self._set_badge(self.lbl_badge_agent, key_name in self._agent_loaded_keys)
        self._set_badge(self.lbl_badge_used, key_name in self._used_keys)
        self._set_badge(self.lbl_badge_tested, bool(self._tested_keys_ok.get(key_name)))

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
                QMessageBox.warning(self, "Key generation", message)

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
            self.btn_mark_used.setText("Unmark as added")
        else:
            self.btn_mark_used.setText("Mark as added to host")

    def _start_agent(self):
        if self._busy:
            return
        self._set_agent_badge("unknown")

        def done(message: str):
            self._log(message)
            self.statusBar().showMessage(message)
            low = message.lower()
            if "could not" in low or "not reachable" in low:
                self._set_agent_badge("off")
                QMessageBox.warning(self, "SSH Agent", message)
            else:
                self._set_agent_badge("on")

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
        marker = "[x]" if done else "[ ]"
        return f"{marker} {text}"

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
