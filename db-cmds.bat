@echo off
REM docker-commands.bat - Windows batch file for Docker Compose operations

IF "%1"=="" GOTO help
IF "%1"=="help" GOTO help
IF "%1"=="up" GOTO up
IF "%1"=="down" GOTO down
IF "%1"=="restart" GOTO restart
IF "%1"=="logs" GOTO logs
IF "%1"=="ps" GOTO ps
IF "%1"=="pg-shell" GOTO pgshell
IF "%1"=="example" GOTO example
IF "%1"=="env" GOTO env

:help
echo Available commands:
echo   db-cmds help       - Show this help message
echo   db-cmds up         - Start all containers (PostgreSQL and Datadog agent)
echo   db-cmds down       - Stop and remove all containers
echo   db-cmds restart    - Restart all containers
echo   db-cmds logs       - Show logs of all containers
echo   db-cmds ps         - Show status of containers
echo   db-cmds pg-shell   - Connect to PostgreSQL shell
echo   db-cmds example    - Run the datadog example
echo   db-cmds env        - Edit the .env file
GOTO end

:up
echo Starting containers...
echo Using environment variables from .env file...
docker compose --env-file .env up -d
GOTO end

:down
echo Stopping containers...
docker compose down
GOTO end

:restart
echo Restarting containers...
docker compose down
echo Using environment variables from .env file...
docker compose --env-file .env up -d
GOTO end

:logs
echo Showing logs...
docker compose logs -f
GOTO end

:ps
echo Showing container status...
docker compose ps
GOTO end

:pgshell
echo Connecting to PostgreSQL shell...
docker compose exec postgres psql -U postgres -d example
GOTO end

:example
echo Running datadog example...
echo Ensuring example database exists...
docker compose exec postgres psql -U postgres -c "CREATE DATABASE example;" 2>NUL || echo Database already exists
echo Loading environment variables from .env file...
cd example\datadog && go run main.go
GOTO end

:env
IF NOT EXIST .env (
    echo Creating default .env file...
    (
        echo # Database Configuration
        echo DB_HOST=localhost
        echo DB_PORT=5432
        echo DB_USER=postgres
        echo DB_PASSWORD=postgres
        echo DB_NAME=example
        echo.
        echo # Datadog Configuration
        echo DD_SERVICE_NAME=db-example
        echo DD_ENV=development
        echo DD_ANALYTICS_RATE=1.0
        echo DD_API_KEY=your_datadog_api_key_here
        echo.
        echo # Other Configuration
        echo # Add any other environment variables your application needs
    ) > .env
)
echo Opening .env file for editing...
notepad .env
GOTO end

:end
