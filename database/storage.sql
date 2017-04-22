-- Storage values
	-- Hallway lockers
	INSERT INTO storage (number, plan_id)
	SELECT generate_series(1,12), 'storage-locker_hallway';
	-- Bathroom lockers
	INSERT INTO storage (number, plan_id)
	SELECT generate_series(1,11), 'storage-locker_bathroom';
		-- Bathroom lockers 7 and 8 are reserved for VITP cleaners
		UPDATE storage
		SET available = false
		WHERE number IN (7, 8)
			AND plan_id = 'storage-locker_bathroom';
	-- Wall storage
	INSERT INTO storage (number, plan_id, quantity)
	SELECT
		generate_subscripts(a, 1),
		'storage-wall_tenth_lineal_foot',
		unnest(a)
	FROM (
		SELECT ARRAY[25,35,30,50,40,50,40,40,40,55] AS a
	) lf;
		-- Storage locations 1 and 2 are owned by makerspace for now
		UPDATE storage
		SET available = false
		WHERE number IN (1, 2)
			AND plan_id = 'storage-wall_tenth_lineal_foot';
