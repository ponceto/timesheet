/*
 * subscriptions_page.js - Copyright (c) 2023-2026 - Olivier Poncet
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

class SubscriptionsPage {
    constructor(document) {
        this.document = document;
        this.window = document.defaultView;
        this.pickers = [...document.querySelectorAll('label.login-picker')];
    }

    start() {
        for (const picker of this.pickers) {
            const field = picker.querySelector('input[name="login"]');
            const list = this.document.createElement('ul');
            list.className = 'login-suggestions';
            list.setAttribute('role', 'listbox');
            list.setAttribute('aria-label', 'Comptes proposés');
            list.hidden = true;
            picker.after(list);
            const state = {timer: 0, request: null};
            field.addEventListener('input', () => this.schedule(field, list, state));
            field.addEventListener('keydown', (event) => this.navigate(event, field, list));
        }
    }

    schedule(field, list, state) {
        this.window.clearTimeout(state.timer);
        state.timer = this.window.setTimeout(() => this.suggest(field, list, state), 300);
    }

    async suggest(field, list, state) {
        const query = field.value.trim();
        if (state.request !== null) {
            state.request.abort();
        }
        if (query.length < 2) {
            this.show(field, list, []);
            return;
        }
        state.request = new AbortController();
        try {
            const response = await this.window.fetch('/dashboard/subscriptions/logins?q=' + encodeURIComponent(query), {credentials: 'same-origin', signal: state.request.signal, headers: {Accept: 'application/json'}});
            const logins = response.ok ? await response.json() : [];
            this.show(field, list, Array.isArray(logins) ? logins : []);
        } catch {
            this.show(field, list, []);
        }
    }

    show(field, list, logins) {
        list.replaceChildren();
        for (const login of logins) {
            const item = this.document.createElement('li');
            item.setAttribute('role', 'option');
            const button = this.document.createElement('button');
            button.type = 'button';
            button.textContent = String(login);
            button.addEventListener('click', () => this.choose(field, list, String(login)));
            button.addEventListener('keydown', (event) => this.navigate(event, field, list));
            item.append(button);
            list.append(item);
        }
        list.hidden = logins.length === 0;
    }

    choose(field, list, login) {
        field.value = login;
        this.show(field, list, []);
        field.focus();
    }

    navigate(event, field, list) {
        const buttons = [...list.querySelectorAll('button')];
        const index = buttons.indexOf(this.document.activeElement);
        if (event.key === 'Escape') {
            this.show(field, list, []);
            field.focus();
        } else if ((event.key === 'ArrowDown') && (buttons.length > 0)) {
            event.preventDefault();
            buttons[Math.min(index + 1, buttons.length - 1)].focus();
        } else if ((event.key === 'ArrowUp') && (index >= 0)) {
            event.preventDefault();
            if (index === 0) {
                field.focus();
            } else {
                buttons[index - 1].focus();
            }
        }
    }
}

new SubscriptionsPage(document).start();
