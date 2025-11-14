# go-musthave-shortener-tpl

GophKeeper представляет собой клиент-серверную систему для безопасного хранения логинов, паролей, бинарных данных и другой приватной информации.

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. Установите зависимости

    ```go mod tidy```

3. Создайте базу данных в PostgreSQL:

    ```CREATE DATABASE gophkeeper;```

4. Создайте пользователя

    ```
    CREATE USER gopher WITH PASSWORD 'gopherpass';
    GRANT ALL PRIVILEGES ON DATABASE gophkeeper TO gopher;
    ```

5. Запуск сервера

    5.1 В режиме разработки на ```http://localhost:8080```:
    ```
    go run cmd/server/main.go
    ```

    5.2 Конфигурация через переменные окружения:
    ``` 
   export DATABASE_URL="postgres://gopher:gopherpass@localhost:5432/gophkeeper?  sslmode=disable"
    export JWT_SECRET="your-secret-key"
    export SERVER_ADDRESS=":8080"
    go run cmd/server/main.go 
    ```

6. Сборка клиента
    ```
    go build -ldflags="-X main.version=1.0.0 -X main.buildDate=$(date +%Y-%m-%d)" -o gophkeeper cmd/client/main.go
    ```
7. Регистрация пользователя
    ```
    ./gophkeeper register myuser mypassword
    ```
8. Вход
    ```
    ./gophkeeper login myuser mypassword
    ```
9. Команды клиента:
    
    Сохранить текстовые данные

    ```echo "my secret data" | ./gophkeeper store --name "My Secret" --type text```

    Сохранить данные из файла

    ```./gophkeeper store --name "Website Login" --type login_password --file credentials.json```

    Сохранить бинарные данные

    ```./gophkeeper store --name "Backup File" --type binary --file backup.tar.gz```

    Просмотреть список данных

    ```./gophkeeper list```

    Получить конкретную запись

    ```./gophkeeper get <entry-id>```

    Обновить запись

    ```echo "updated data" | ./gophkeeper update <entry-id> --name "Updated Name"```

    Удалить запись

    ```./gophkeeper delete <entry-id>```



10. Безопасность

    10.1 Сгенерируйте тестовые сертификаты:

    ```
    mkdir -p tls
    openssl req -x509 -newkey rsa:4096 -keyout tls/server.key -out tls/server.crt -days 365 -nodes -subj "/CN=localhost"
    ```

    10.2 Запустите сервер с TLS:
    ```
    export ENABLE_TLS=true
    export TLS_CERT_FILE=tls/server.crt
    export TLS_KEY_FILE=tls/server.key
    export SERVER_ADDRESS=:8443
    go run cmd/server/main.go
    ```
    10.3 Используйте клиент с HTTPS:

    ```
    export GOPHKEEPER_SERVER=https://localhost:8443
    export GOPHKEEPER_SKIP_VERIFY=true
    ./gophkeeper login myuser mypassword
    ```


    ### Требования к логину и паролю

**Логин:**
- Минимум 3 символа, максимум 50 символов
- Должен начинаться с буквы
- Может содержать только: буквы (a-z, A-Z), цифры (0-9), подчеркивание (_), дефис (-)
- Не может содержать пробелы или специальные символы

**Пароль:**
- Минимум 8 символов, максимум 72 символа
- Должен содержать хотя бы одну заглавную букву
- Должен содержать хотя бы одну строчную букву  
- Должен содержать хотя бы одну цифру
- Должен содержать хотя бы один специальный символ (!@#$%^&* и т.д.)
- Не может быть распространенным паролем (password, 12345678 и т.д.)

