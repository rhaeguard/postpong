package main

import (
	"context"
	"database/sql"
	"log"

	rl "github.com/gen2brain/raylib-go/raylib"
	_ "github.com/lib/pq"
)

const GAP = 12.0
const (
	ScreenWidth  = 1280
	ScreenHeight = 720
)

func exec(conn *sql.Conn, functionCreationSql string, args ...any) {
	if _, err := conn.ExecContext(
		context.Background(),
		functionCreationSql, args...,
	); err != nil {
		log.Fatalf("(%s)\n%s\n", functionCreationSql, err)
	} else {
	}
}

func ticker(conn *sql.Conn) {
	rows, err := conn.QueryContext(context.Background(), `SELECT update();`)
	if err != nil {
		panic(err.Error())
	}
	rows.Close()
}

func main() {
	screen := rl.NewRectangle(
		0, 0, float32(ScreenWidth), float32(ScreenHeight),
	)
	playableBorder := rl.NewRectangle(
		GAP, GAP, screen.Width-(2*GAP), screen.Height-(2*GAP),
	)

	top := rl.NewRectangle(
		playableBorder.X, screen.Y, playableBorder.Width, playableBorder.Y,
	)

	bottom := rl.NewRectangle(
		playableBorder.X, playableBorder.Y+playableBorder.Height, playableBorder.Width, playableBorder.Y,
	)
	racketWidth := float32(200)

	// database stuff
	db, _ := sql.Open("postgres", "postgres://postgres:mysecretpassword@127.0.0.1:5432/postgres?sslmode=disable")
	if err := db.Ping(); err != nil {
		log.Fatalf("%s\n", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	defer conn.Close()

	{ // init code
		functionCreationSql := `
drop table if exists user_inputs;
create table user_inputs(
		player_id int,
		move int,
		x real, -- user properties
		y real,
		b_x real, -- ball properties
		b_y real,
		b_r real,
		b_vx real, -- ball velocity properties
		b_vy real
);

drop table if exists flat_table;
create table flat_table(
		id int,
		screen_x real,
		screen_y real,
		screen_w real,
		screen_h real,
		top_x real,
		top_y real,
		top_w real,
		top_h real,
		bottom_x real,
		bottom_y real,
		bottom_w real,
		bottom_h real,
		racket_width real
);

create or replace function get_state()
   returns table (
		ox real,
		oy real
   )
   language plpgsql
  as
$$
declare
   ox real;
   oy real;
begin
	select x, y into ox, oy from user_inputs;
	return query select ox, oy;
end;
$$;

create or replace function check_collision_circle_rec(
		cx real,
		cy real,
		cr real,
		rx real,
		ry real,
		rw real,
		rh real
)
   returns bool
   language plpgsql
  as
$$
declare
	collision bool;
	recCenterX real;
	recCenterY real;
	dx real;
	dy real;
	corderDistSq real;
begin
		recCenterX := rx + rw/2.0;
		recCenterY := ry + rh/2.0;

		dx := abs(cx - recCenterX);
		dy := abs(cy - recCenterY);

		if dx > (rw/2.0) + cr then
			return false;
		end if;

		if dy > (rh/2.0) + cr then
			return false;
		end if;

		if dx <= (rw/2.0) then
			return true;
		end if;

		if dy <= (rh/2.0) then
			return true;
		end if;

		corderDistSq := (dx - (rw/2.0))*(dx - (rw/2.0)) + (dy - (rh/2.0))*(dy - (rh/2.0)); 

		collision := corderDistSq <= cr*cr;

	return collision;
end;
$$;

create or replace function update()
   returns int
   language plpgsql
  as
$$
declare
   ix real default -1;
   iy real default -1;
   ox real default -1;
   oy real default -1;
   racketWidth real default -1;
   racketPositionY real;
   screenHeight real default -1;
   user_move int;
   bx real;
   by_ real;
   br real;
   bvx real;
   bvy real;
   i_bottom_x real;
   i_bottom_y real;
   i_bottom_w real;
   i_bottom_h real;
   bottom_check bool;
   i_top_x real;
   i_top_y real;
   i_top_w real;
   i_top_h real;
   top_check bool;
   right_check bool;
   racket_check bool;
begin
	select 
		x, y, b_x, b_y, b_r, b_vx, b_vy,
		move
		into 
		ix, iy, bx, by_, br, bvx, bvy,
		user_move 
	from user_inputs where player_id=1;
	
	select 
		screen_h, racket_width 
		into screenHeight, racketWidth 
	from flat_table where id=1;

	if user_move = 1 then
		oy := iy - 10; 
	elsif user_move = 2 then
		oy := iy + 10;
	else
		ox := ix;
		oy := iy;
	end if;

	if oy - racketWidth/2 < 0 then
		oy := iy;
	elsif oy + racketWidth/2 > screenHeight then
		oy := iy;
	end if;

	update user_inputs set x = ix, y = oy where player_id=1;

	-- update the ball starts
	bx := bx + bvx;
	by_ := by_ + bvy;

	select 
		bottom_x, bottom_y, bottom_w, bottom_h,
		top_x, top_y, top_w, top_h
	into
		i_bottom_x, i_bottom_y, i_bottom_w, i_bottom_h,
		i_top_x, i_top_y, i_top_w, i_top_h
	from flat_table where id=1;

	bottom_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		i_bottom_x, 
		i_bottom_y, 
		i_bottom_w, 
		i_bottom_h
	);

	top_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		i_top_x, 
		i_top_y, 
		i_top_w, 
		i_top_h
	);

	right_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		i_top_x+i_top_w, 
		i_top_y, 
		i_top_h, 
		i_top_w
	);

	racketPositionY := oy - racketWidth/2;

	racket_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		ox, -- assuming player on the left 
		racketPositionY, 
		i_top_y, 
		racketWidth
	);

	if bottom_check then
		update user_inputs set b_vy = (-bvy) where player_id=1;
	end if;

	if top_check then
		update user_inputs set b_vy = (-bvy) where player_id=1;
	end if;

	if right_check then
		update user_inputs set b_vx = (-bvx) where player_id=1;
	end if;

	if racket_check then
		update user_inputs set b_vx = (-bvx) where player_id=1;
	end if;

	
	
	update user_inputs set b_x = bx, b_y = by_ where player_id=1;
	-- update the ball ends

	return 1;
end;
$$;
`
		exec(conn, functionCreationSql)

		exec(
			conn, `insert into flat_table values (1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);`,
			// args
			screen.X, screen.Y, screen.Width, screen.Height,
			top.X, top.Y, top.Width, top.Height,
			bottom.X, bottom.Y, bottom.Width, bottom.Height,
			racketWidth,
		)

		exec(
			conn, `insert into user_inputs values (1, 0, $1, $2, $3, $4, $5, $6, $7);`,
			// args
			0.0, screen.Height/2,
			screen.Width/2, screen.Height/2,
			screen.Width/128,
			10, 10,
		)
	}

	rl.SetTraceLogLevel(rl.LogError)
	rl.InitWindow(int32(screen.Width), int32(screen.Height), "bingbong")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	var x float32
	var y float32

	// ball properties
	var bx, by, br float32

	for !rl.WindowShouldClose() {
		{
			move := 0
			// grab user input section
			if rl.IsKeyDown(rl.KeyUp) {
				move = 1
			}

			if rl.IsKeyDown(rl.KeyDown) {
				move = 2
			}

			exec(conn, `UPDATE user_inputs SET move=$1 WHERE player_id=1;`, move)
		}
		{
			// update section
			ticker(conn)
		}
		{
			rows, err := conn.QueryContext(context.Background(), `SELECT x, y, b_x, b_y, b_r from user_inputs;`)
			if err != nil {
				panic(err.Error())
			}

			for rows.Next() {
				if err := rows.Scan(&x, &y, &bx, &by, &br); err != nil {
					panic(err.Error())
				}
			}

			rows.Close()
		}

		rl.BeginDrawing()

		rl.ClearBackground(rl.RayWhite)

		rl.DrawRectangleV(
			rl.NewVector2(screen.X, y-racketWidth/2),
			rl.NewVector2(GAP, racketWidth),
			rl.Blue,
		)
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
}
