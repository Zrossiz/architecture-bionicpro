Добавляем idp keycloack
Настраиваем federation/identity brokering для подключения внешних устройств
Это будет как SSO для всех приложений (crm, интернет магазин и приложение для донастройки протеза)
Все пользователи получают только access token. Для получения access + refresh token (refresh token хранится на сервере и клиент его не получает) нужно передать auth code + code verifier. Бекенд обновляет access token по refresh token
Для безопасности предлагаю добавить RBAC - и для каждой роли прописать свои пермишны. В access token можно положить название роли.
Персональные данные хранятся только в локальных бд. Keycloack хранит в себе только идентификаторы пользователя и тех. атрибуты для SSO
Для включения PKCE
1. Запуск docker compose
2. Ввод логина и пароля
3. Clients (сайдбар)
4. Client details
5. Advanced
6. Advanced Settings
7. Proof Key for Code Exchange Code Challenge Method S256