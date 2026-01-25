meowpick-run:
	@echo "初始化 meowpick 数据库和缓存服务"
	docker-compose --project-name meowpick -f ./docker-compose.yml up -d meowpick-mongodb meowpick-redis
	@echo "初始化完成"

meowpick-clean:
	@echo "删除 meowpick 数据库和缓存服务"
	docker-compose --project-name meowpick -f ./docker-compose.yml down --remove-orphans
	@echo "删除完成"