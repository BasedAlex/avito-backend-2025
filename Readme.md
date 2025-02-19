# Тестовое задание для стажера Backend

  

## Описание задачи

  

Необходимо реализовать сервис, который позволит сотрудникам обмениваться монетками и приобретать на них мерч. Каждый сотрудник должен иметь возможность видеть:

  

1) Список купленных им мерчовых товаров

2) Сгруппированную информацию о перемещении монеток в его кошельке, включая:

	а) Кто ему передавал монетки и в каком количестве

	b) Кому сотрудник передавал монетки и в каком количестве


Количество монеток не может быть отрицательным, запрещено уходить в минус при операциях с монетками.

  

## Инструкция по запуску

  

В репозитории находится Taskfile и Makefile для удобства, команды в них работают идентично, буду описывать сценарий запуска через Make.

  

- Для первого запуска необходимо проинициализировать проект и базу данных через команду `make build`

- Затем запустить приложение через команду `make up`

- В дальнейшем работа с приложением ведётся через команды `make up` для старта сервера и `make restart` для рестарта сервера

- для запуска юнит тестов использовать команду `make test`

- для запуска интеграционных тестов `make integration`

- для запуска линтера `make lint`

  

По умолчанию сервер доступен по адресу `http://localhost:8080/`

При первой миграции добавляются товары в merch_shop, чтобы облегчить тестирование приложения.

**Мерч** — это продукт, который можно купить за монетки. Всего в магазине доступно 10 видов мерча. Каждый товар имеет уникальное название и цену. Ниже приведён список наименований и их цены.

| Название     | Цена |
|--------------|------|
| t-shirt      | 80   |
| cup          | 20   |
| book         | 50   |
| pen          | 10   |
| powerbank    | 200  |
| hoody        | 300  |
| umbrella     | 200  |
| socks        | 10   |
| wallet       | 50   |
| pink-hoody   | 500  |
  

# Описание эндпоинтов

  

## (POST /api/auth)

  

Это эндпоинт для авторизации и/или регистрации. Если пользователь из запроса существует в системе с корректным логином и паролем он авторизуется, если такого пользователя не существует он будет создан.
  

В теле запроса ожидается `username` и `password`

  

type AuthRequest struct {

		Password string `json:"password"`

		Username string `json:"username"`

}

  

## Response

  

В случае успешного ответа сервер вернёт 200 статус код и JWT токен для дальнейшей авторизации. В дальнейшем используется Authorization Bearer `token`.

  

## (GET /api/buy/{item})

  

Это ендпоинт для покупки мерч-товаров, в URL ожидается название мерч-товара, который будет куплен. Данный эндпоинт защищён авторизацией, поэтому имя сотрудника будет получено из его JWT токена.

  

## Response

  

В случае успешного ответа сервер вернёт 200 статус код.

  

## (GET /api/info)

  

Это ендпоинт для просмотра истории покупок и переводов, которые совершал или получал сотрудник. Данный эндпоинт защищён авторизацией, поэтому имя сотрудника будет получено из его JWT токена.

  

## Response

  

В случае успешного ответа сервер вернёт 200 статус код и тело ответа.

  

type InfoResponse struct {

		Coins int `json:"coins"`

		Inventory []Item `json:"inventory"`

		CoinHistory CoinHistory `json:"coinHistory"`

}

  

type Item struct {
	
		Type string `json:"type"`

		Quantity int `json:"quantity"`

}

  

type CoinHistory struct {

		Received []Transaction `json:"received"`

		Sent []Transaction `json:"sent"`

}

  

type Transaction struct {

		FromUser string `json:"fromUser,omitempty"`

		ToUser string `json:"toUser,omitempty"`

		Amount int `json:"amount"`

		TransactionDate time.Time `json:"transactionDate,omitempty"`

}

  
  

## (POST /api/sendCoin)


Это ендпоинт для переводов другим сотрудникам. Данный эндпоинт защищён авторизацией, поэтому имя сотрудника будет получено из его JWT токена. В теле запроса ожидается информация о количестве монет и получателе перевода.

  

type SendCoinRequest struct {
	
		Amount int `json:"amount"`

		ToUser string `json:"toUser"`

}

  

## Response

В случае успешного ответа сервер вернёт 200 статус код.