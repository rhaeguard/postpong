# postpong

pingpong but with postgres.

The gameplay so far (played in two different machines (Windows + Linux) in the same local network):

[https://github.com/user-attachments/assets/c0324554-52d3-4423-b0d0-136fadfee9fc](https://github.com/user-attachments/assets/cbbe33a9-48e5-4d9d-b58f-82998034d826)

Currently the architecture is as follows:
- There's a postgres database that is responsible for holding the game state and updating it.
  - an example instance can be run using docker: `docker run --name some-postgres -e POSTGRES_PASSWORD=mysecretpassword -d -p 5432:5432 postgres`
- There's a game server that is responsible for running the game (basically triggering the `update` function)
  - this can be done by running the game executable using `-server` param.
- Currently the game has no matchmaking, so there's only one "match". As players connect, they pick an available slot (either player A or B)
  - this can be done by running the game executable

The executables need to be run with these environment variables set:
- `PONG_PG_USERNAME`
- `PONG_PG_PASSWORD`
- `PONG_PG_NETLOC`
- `PONG_PG_PORT`
- `PONG_PG_DB`

### but why?

The idea is to put the main processing logic of the game in the Postgres db by utilizing PL/PGSQL functions. Then the results of the process can be retrieved by the game and directly rendered. This is what's happening in this pong example. The idea is that maybe if we write our game logic in the database itself, it'll be easier to turn a game into a multiplayer? Got inspired by [this SpacetimeDB talk](https://www.youtube.com/watch?v=kzDnA_EVhTU).
