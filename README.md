# postpong

pingpong but with postgres.

https://github.com/user-attachments/assets/c0324554-52d3-4423-b0d0-136fadfee9fc

```sh
# example postgres instance
docker run --name some-postgres -e POSTGRES_PASSWORD=mysecretpassword -d -p 5432:5432 postgres
# from the source dir.
go run .
```

The idea is to put the main processing logic of the game in the Postgres db by utilizing PL/PGSQL functions. Then the results of the process can be retrieved by the game and directly rendered. This is what's happening in this pong example. The idea is that maybe if we write our game logic in the database itself, it'll be easier to turn a game into a multiplayer?
