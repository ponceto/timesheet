/*
 * assistant_page.js - Copyright (c) 2023-2026 - Olivier Poncet
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

class AssistantPage {
    constructor(document) {
        this.forms = [...document.querySelectorAll('form.question-form')];
        this.exchanges = [...document.querySelectorAll('article.exchange')];
    }

    start() {
        for (const form of this.forms) {
            const field = form.querySelector('textarea[name="question"]');
            if (field !== null) {
                field.addEventListener('keydown', (event) => this.send(event, form));
            }
            form.addEventListener('submit', () => this.lock(form));
        }
        if (this.exchanges.length > 0) {
            this.exchanges[this.exchanges.length - 1].scrollIntoView({block: 'start'});
        }
    }

    send(event, form) {
        if ((event.key === 'Enter') && !event.shiftKey && !event.isComposing) {
            event.preventDefault();
            form.requestSubmit();
        }
    }

    lock(form) {
        const button = form.querySelector('button[type="submit"]');
        if (button !== null) {
            button.disabled = true;
            button.textContent = 'Envoi…';
        }
    }
}

new AssistantPage(document).start();
