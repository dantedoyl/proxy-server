# proxy-server

### Запуск
```bigquery
sudo docker build . -t proxy-server
sudo docker run -p 8080:8080 --name proxy proxy-server
```

### Проксирование
```bigquery
curl -x http://localhost:8080/ http://mail.ru
```

### Список запросов
```bigquery
http://localhost:8080/requests
```

### Информация о запросе по id
```bigquery
http://localhost:8080/request/1
```

### Информация о запросе по id
```bigquery
http://localhost:8080/repeat/1
```

### Param-miner
```bigquery
http://localhost:8080/scan/1
```