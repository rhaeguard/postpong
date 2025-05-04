package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	_ "github.com/lib/pq"
)

const GAP = 12.0
const (
	ScreenWidth  = 1280
	ScreenHeight = 720
)

func exec(conn *sql.Conn, sql string, args ...any) {
	if _, err := conn.ExecContext(
		context.Background(),
		sql, args...,
	); err != nil {
		log.Fatalf("(%s)\n%s\n", sql, err)
	} else {
	}
}

type ExecPair struct {
	name  string
	value any
}

func ep(name string, value any) ExecPair {
	return ExecPair{
		name, value,
	}
}

func execWithNamedArgs(conn *sql.Conn, sql string, args ...ExecPair) {
	sqlPlaceholders := []string{}
	sqlArgs := []any{}
	for i, pair := range args {
		sqlPlaceholders = append(sqlPlaceholders, fmt.Sprintf("$%d", i+1))
		sqlArgs = append(sqlArgs, pair.value)
	}
	placeholder := strings.Join(sqlPlaceholders, ",")
	sql = strings.Replace(sql, "...", placeholder, 1)

	if _, err := conn.ExecContext(context.Background(), sql, sqlArgs...); err != nil {
		log.Fatalf("(%s)\n%s\n", sql, err)
	} else {
	}
}

var screen rl.Rectangle
var playableBorder rl.Rectangle
var top rl.Rectangle
var bottom rl.Rectangle
var racketWidth float32
var racketThickness float32

func initialize() {
	screen = rl.NewRectangle(
		0, 0, float32(ScreenWidth), float32(ScreenHeight),
	)
	playableBorder = rl.NewRectangle(
		GAP, GAP, screen.Width-(2*GAP), screen.Height-(2*GAP),
	)

	top = rl.NewRectangle(
		playableBorder.X, screen.Y, playableBorder.Width, playableBorder.Y,
	)

	bottom = rl.NewRectangle(
		playableBorder.X, playableBorder.Y+playableBorder.Height, playableBorder.Width, playableBorder.Y,
	)
	racketWidth = float32(200)
	racketThickness = GAP
}

func initializeDB(conn *sql.Conn) {
	bytes, err := os.ReadFile("init.sql")

	if err != nil {
		log.Fatalf("Could not open the init.sql file: %s", err.Error())
	}

	functionCreationSql := string(bytes)

	exec(conn, functionCreationSql)

	execWithNamedArgs(
		conn, `insert into flat_table values (...);`,
		ep("id", 1),
		// screen
		ep("screen_x", screen.X),
		ep("screen_y", screen.Y),
		ep("screen_w", screen.Width),
		ep("screen_h", screen.Height),
		// top border
		ep("top_x", top.X),
		ep("top_y", top.Y),
		ep("top_w", top.Width),
		ep("top_h", top.Height),
		// bottom border
		ep("bottom_x", bottom.X),
		ep("bottom_y", bottom.Y),
		ep("bottom_w", bottom.Width),
		ep("bottom_h", bottom.Height),
		// racket
		ep("racket_width", racketWidth),
		ep("racket_thickness", racketThickness),
	)

	execWithNamedArgs(
		conn, `insert into user_inputs values (...);`,
		ep("game_id", 1),
		// player a
		ep("player_a_id", 1),
		ep("player_a_move", 0),
		ep("player_a_x", 0.0),
		ep("player_a_y", screen.Height/2),
		ep("player_a_active", false),
		// player b
		ep("player_b_id", 1),
		ep("player_b_move", 0),
		ep("player_b_x", screen.Width),
		ep("player_b_y", screen.Height/2),
		ep("player_b_active", false),
		// ball position
		ep("b_x", screen.Width/2),
		ep("b_y", screen.Height/2),
		ep("b_r", screen.Width/128),
		ep("b_vx", 5), // TODO: ball speed?
		ep("b_vy", 5),
	)
}

func update(conn *sql.Conn) {
	const TickRate = 60

	ticker := time.NewTicker(time.Second / TickRate)

	for range ticker.C {
		exec(conn, `SELECT update();`) // 16.67 ms hard limit
	}
}

type Player struct {
	id    int
	move  int
	x     float32
	y     float32
	color rl.Color
}

func gameplay(conn *sql.Conn) {
	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(int32(screen.Width), int32(screen.Height), "postpong")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// ball properties
	var bx, by, br float32

	total := float32(0.0)
	totalCount := 0

	var playerA = Player{
		id:   1,
		move: 0,
		x:    0.0,
		y:    0.0,
	}

	var playerB = Player{
		id:   1,
		move: 0,
		x:    0.0,
		y:    0.0,
	}

	var myPlayer = ""

	{
		var playerAActive, playerBActive bool
		rows, err := conn.QueryContext(
			context.Background(),
			`SELECT player_a_active, player_b_active from user_inputs where game_id=1;`,
		)
		if err != nil {
			panic(err.Error())
		}

		for rows.Next() {
			if err := rows.Scan(&playerAActive, &playerBActive); err != nil {
				panic(err.Error())
			}
		}
		rows.Close()

		if !playerAActive && !playerBActive {
			myPlayer = "player_a"
		} else if playerAActive && playerBActive {
			panic("no slot available, try later!")
		} else if playerAActive && !playerBActive {
			myPlayer = "player_b"
		} else if !playerAActive && playerBActive {
			myPlayer = "player_a"
		}

		if myPlayer == "player_a" {
			exec(conn, `UPDATE user_inputs SET player_a_active=TRUE WHERE player_a_id=$1 AND game_id=1;`, playerA.id)
		} else {
			exec(conn, `UPDATE user_inputs SET player_b_active=TRUE WHERE player_b_id=$1 AND game_id=1;`, playerB.id)
		}

	}

	var player *Player

	if myPlayer == "player_a" {
		player = &playerA
	} else {
		player = &playerB
	}

	for !rl.WindowShouldClose() {
		start := time.Now()
		{
			{
				newMove := 0
				// grab user input section
				if rl.IsKeyDown(rl.KeyUp) {
					newMove = 1
				}

				if rl.IsKeyDown(rl.KeyDown) {
					newMove = 2
				}

				if player.move != newMove {
					sql := `UPDATE user_inputs SET player_b_move=$1 WHERE player_b_id=$2 AND game_id=1;`
					if myPlayer == "player_a" {
						sql = `UPDATE user_inputs SET player_a_move=$1 WHERE player_a_id=$2 AND game_id=1;`
					}
					exec(conn, sql, newMove, player.id)
					player.move = newMove
				}
			}
		}
		{
			rows, err := conn.QueryContext(
				context.Background(),
				`SELECT 
				player_a_x, player_a_y, 
				player_b_x, player_b_y, 
				b_x, b_y, b_r 
				from user_inputs 
				where game_id=1;`,
			)
			if err != nil {
				panic(err.Error())
			}

			for rows.Next() {
				if err := rows.Scan(&playerA.x, &playerA.y, &playerB.x, &playerB.y, &bx, &by, &br); err != nil {
					panic(err.Error())
				}
			}

			rows.Close()
		}
		total += float32(time.Since(start))
		totalCount++

		rl.BeginDrawing()

		rl.ClearBackground(rl.RayWhite)

		// draw player A
		rl.DrawRectangleV(
			rl.NewVector2(screen.X, playerA.y-racketWidth/2),
			rl.NewVector2(GAP, racketWidth),
			rl.Blue,
		)
		// draw player B
		rl.DrawRectangleV(
			rl.NewVector2(screen.Width-GAP, playerB.y-racketWidth/2),
			rl.NewVector2(GAP, racketWidth),
			rl.DarkGreen,
		)
		// draw ball
		rl.DrawCircleV(
			rl.NewVector2(bx, by),
			br,
			rl.Red,
		)

		rl.DrawRectangleLinesEx(
			playableBorder, 1, rl.Black,
		)

		rl.DrawRectangleRec(top, rl.Red)
		rl.DrawRectangleRec(bottom, rl.Red)

		rl.DrawFPS(0, 0)

		rl.EndDrawing()
	}

	if myPlayer == "player_a" {
		exec(conn, `UPDATE user_inputs SET player_a_active=FALSE WHERE player_a_id=$1 AND game_id=1;`, playerA.id)
	} else {
		exec(conn, `UPDATE user_inputs SET player_b_active=FALSE WHERE player_b_id=$1 AND game_id=1;`, playerB.id)
	}

	avg := (total / float32(totalCount)) / 1e6
	fmt.Printf("Avg Update Time: %.4f\n", avg)
}

func GetEnvOrDefault(varname string, defaultValue string) string {
	if value, ok := os.LookupEnv(varname); ok {
		return value
	}
	return defaultValue
}

func main() {
	isServer := flag.Bool("server", false, "is this binary the server or the game client")
	flag.Parse()

	var pgUsername string = GetEnvOrDefault("PONG_PG_USERNAME", "postgres")
	var pgPasswd string = GetEnvOrDefault("PONG_PG_PASSWORD", "mysecretpassword")
	var pgNetloc string = GetEnvOrDefault("PONG_PG_NETLOC", "127.0.0.1")
	var pgPort string = GetEnvOrDefault("PONG_PG_PORT", "5432")
	var pgDb string = GetEnvOrDefault("PONG_PG_DB", "postgres")

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgUsername, pgPasswd, pgNetloc, pgPort, pgDb,
	)

	// database stuff
	db, _ := sql.Open("postgres", connectionString)
	if err := db.Ping(); err != nil {
		log.Fatalf("%s\n", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	defer conn.Close()

	initialize()

	if *isServer {
		fmt.Println("Server is running...")
		initializeDB(conn)
		update(conn)
	} else {
		fmt.Println("Game is running...")
		gameplay(conn)
	}
}
