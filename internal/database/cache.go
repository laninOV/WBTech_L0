package database

import (
	"log"
	"os"
	"strconv"
	"sync"
)

// Cache представляет структуру для кэширования данных.
type Cache struct {
	buffer  map[string]interface{} // Буфер кэша
	queue   []string               // Очередь элементов кэша
	bufSize int                    // Размер кэша
	pos     int                    // Текущая позиция в очереди
	DBInst  *DB                    // Экземпляр базы данных
	name    string                 // Имя кэша
	mutex   *sync.RWMutex          // Мьютекс для синхронизации доступа к кэшу
}

// NewCache создает новый экземпляр кэша.
func NewCache(db *DB) *Cache {
	csh := Cache{
		DBInst: db,
		name:   "Cache",
		mutex:  &sync.RWMutex{},
	}
	csh.init()
	return &csh
}

// init выполняет инициализацию кэша.
func (c *Cache) init() {
	db := c.DBInst
	db.SetCacheInstance(c)
	c.bufSize = c.getCacheSize()
	c.buffer = make(map[string]interface{}, c.bufSize)
	c.queue = make([]string, c.bufSize)
	c.pos = 0
	c.restoreFromDatabase()
}

// getCacheSize получает размер кэша из переменных окружения.
func (c *Cache) getCacheSize() int {
	bufSize, err := strconv.Atoi(os.Getenv("CACHE_SIZE"))
	if err != nil {
		log.Printf("%s: Предупреждение: Установлен размер кэша по умолчанию - 10\n", c.name)
		bufSize = 10
	}
	return bufSize
}

// restoreFromDatabase восстанавливает данные кэша из базы данных.
func (c *Cache) restoreFromDatabase() {
	log.Printf("%s: Проверка и загрузка кэша из базы данных\n", c.name)
	buf, queue, pos, err := c.DBInst.GetCacheState(c.bufSize)
	if err != nil {
		log.Printf("%s: Предупреждение: Не удалось загрузить из базы данных или кэш пуст: %v\n", c.name, err)
		return
	}

	c.mutex.Lock()
	copy(c.queue, queue)
	c.pos = pos
	c.buffer = buf
	c.mutex.Unlock()

	log.Printf("%s: Кэш загружен из базы данных: Очередь: %v, Следующая позиция в очереди: %v", c.name, c.queue, c.pos)
}

// Set добавляет данные в кэш.
func (c *Cache) Set(key string, value interface{}) {
	if c.bufSize == 0 {
		log.Printf("%s: Кэш отключен: bufSize = 0 (см. config.go)\n", c.name)
		return
	}

	c.mutex.Lock()
	c.queue[c.pos] = key
	c.pos++
	if c.pos == c.bufSize {
		c.pos = 0
	}
	c.buffer[key] = value
	c.mutex.Unlock()

	c.DBInst.SendOrderIDToCache(key)
	log.Printf("%s: Данные успешно добавлены в кэш, Позиция в очереди: %v\n", c.name, c.pos)
}

// Get получает данные из кэша по ключу.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	data, exists := c.buffer[key]
	c.mutex.RUnlock()
	return data, exists
}

// Finish завершает работу кэша и очищает его содержимое в базе данных.
func (c *Cache) Finish() {
	log.Printf("%s: Завершение работы...", c.name)
	c.DBInst.ClearCache()
	log.Printf("%s: Завершено", c.name)
}
