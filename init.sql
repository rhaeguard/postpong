drop table if exists user_inputs;

create table user_inputs(
        game_id int primary key,
        -- player a props
		player_a_id int,
		player_a_move int,
		player_a_x real, -- user properties
		player_a_y real,
		player_a_active bool,
        -- player b props
		player_b_id int,
        player_b_move int,
		player_b_x real, -- user properties
		player_b_y real,
		player_b_active bool,
        -- game props
		b_x real, -- ball properties
		b_y real,
		b_r real,
		b_vx real, -- ball velocity properties
		b_vy real
);

drop table if exists flat_table;
create table flat_table(
		id int primary key,
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
		racket_width real,
        racket_thickness real
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
   -- player a
   p_a_ix real default -1;
   p_a_iy real default -1;
   p_a_ox real default -1;
   p_a_oy real default -1;
   p_a_move int;
   p_a_active bool;
   -- player b
   p_b_ix real default -1;
   p_b_iy real default -1;
   p_b_ox real default -1;
   p_b_oy real default -1;
   p_b_move int;
   p_b_active bool;
   -- other stuff
   racketWidth real default -1;
   racketThickness real default -1;
   racketPositionY real;
   screenHeight real default -1;
   -- ball
   bx real;
   by_ real;
   br real;
   bvx real;
   bvy real;
   -- borders
   i_bottom_x real;
   i_bottom_y real;
   i_bottom_w real;
   i_bottom_h real;
   i_top_x real;
   i_top_y real;
   i_top_w real;
   i_top_h real;
   -- checks
   top_check bool;
   bottom_check bool;
   p_a_racket_check bool;
   p_b_racket_check bool;
begin
	-- should we begin the game?
	select
		player_a_active, player_b_active
	into
		p_a_active, p_b_active
	from user_inputs where game_id=1;

	if not p_a_active or not p_b_active then
		return 2;
	end if;

	select 
        -- player a
		player_a_x, player_a_y, player_a_move,
        -- player b
		player_b_x, player_b_y, player_b_move,
        -- ball fields
        b_x, b_y, b_r, b_vx, b_vy
    into 
		p_a_ix, p_a_iy, p_a_move, -- player a
		p_b_ix, p_b_iy, p_b_move, -- player b
        bx, by_, br, bvx, bvy     -- ball
	from user_inputs where game_id=1;
	
	select 
		screen_h, racket_width, racket_thickness
		into screenHeight, racketWidth, racketThickness
	from flat_table where id=1;

    -- player a
	if p_a_move = 1 then
		p_a_oy := p_a_iy - 10; 
	elsif p_a_move = 2 then
		p_a_oy := p_a_iy + 10;
	else
		p_a_ox := p_a_ix;
		p_a_oy := p_a_iy;
	end if;
    
	if p_a_oy - racketWidth/2 < 0 then
		p_a_oy := p_a_iy;
	elsif p_a_oy + racketWidth/2 > screenHeight then
		p_a_oy := p_a_iy;
	end if;
    -- player a end

    -- player b start
	if p_b_move = 1 then
		p_b_oy := p_b_iy - 10; 
	elsif p_b_move = 2 then
		p_b_oy := p_b_iy + 10;
	else
		p_b_ox := p_b_ix;
		p_b_oy := p_b_iy;
	end if;

    if p_b_oy - racketWidth/2 < 0 then
		p_b_oy := p_b_iy;
	elsif p_b_oy + racketWidth/2 > screenHeight then
		p_b_oy := p_b_iy;
	end if;
    -- player b end

	update user_inputs 
    set 
        player_a_x = p_a_ix, player_a_y = p_a_oy, 
        player_b_x = p_b_ix, player_b_y = p_b_oy 
    where game_id=1; -- update the user positions

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

    -- racket checks
	p_a_racket_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		p_a_ox,
		CAST((p_a_oy - racketWidth/2) as real), 
		racketThickness, 
		racketWidth
	);

    p_b_racket_check := check_collision_circle_rec(
		bx,
		by_,
		br, 
		p_b_ox - racketThickness,
		CAST((p_b_oy - racketWidth/2) as real), 
		racketThickness, 
		racketWidth
	);

	if bottom_check then
		update user_inputs set b_vy = (-bvy) where game_id=1;
	end if;

	if top_check then
		update user_inputs set b_vy = (-bvy) where game_id=1;
	end if;

	if p_a_racket_check or p_b_racket_check then
		update user_inputs set b_vx = (-bvx) where game_id=1;
	end if;
	
	update user_inputs set b_x = bx, b_y = by_ where game_id=1;
	-- -- update the ball ends

	return 1;
end;
$$;