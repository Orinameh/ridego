package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ridego/services/location/internal/models"
)

const (
	geoKey     = "ridego:drivers:geo"    // GEO set — all active drivers
	metaKeyFmt = "ridego:driver:%s:meta" // Hash — heading, speed, updated_at
	onlineKey  = "ridego:drivers:online" // SET   — currently online driver IDs
	ttlOnline  = 90 * time.Second        // driver marked offline if no ping for 90s
)

type GeoStore interface {
	UpdateLocation(ctx context.Context, inp models.UpdateLocationInput) error
	NearbyDrivers(ctx context.Context, req models.NearbyRequest) ([]models.NearbyDriver, error)
	GetDriverLocation(ctx context.Context, driverID string) (*models.DriverLocation, error)
	SetOnline(ctx context.Context, driverID string) error
	SetOffline(ctx context.Context, driverID string) error
	IsOnline(ctx context.Context, driverID string) (bool, error)
}

type RedisGeo struct{ rdb *redis.Client }

func NewRedisGeo(rdb *redis.Client) GeoStore { return &RedisGeo{rdb: rdb} }

// UpdateLocation writes the driver's position atomically using a pipeline.
// Three Redis calls, one round-trip: GEOADD + HSET meta + EXPIRE meta.
func (g *RedisGeo) UpdateLocation(ctx context.Context, inp models.UpdateLocationInput) error {
	metaKey := fmt.Sprintf(metaKeyFmt, inp.DriverID)

	_, err := g.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// GEOADD stores lon/lat in a sorted set using the Geohash encoding
		pipe.GeoAdd(ctx, geoKey, &redis.GeoLocation{
			Name:      inp.DriverID,
			Longitude: inp.Lng,
			Latitude:  inp.Lat,
		})
		// Meta hash stores additional fields not held in GEO set
		pipe.HSet(ctx, metaKey,
			"heading", inp.Heading,
			"speed_kmh", inp.SpeedKmh,
			"updated_at", time.Now().UTC().Format(time.RFC3339Nano),
		)
		// Expire meta hash — if driver goes offline the meta cleans itself up
		pipe.Expire(ctx, metaKey, ttlOnline*2)
		return nil
	})
	return err
}

// NearbyDrivers uses GEORADIUS (via GeoSearchStore) to find active drivers.
// Returns results sorted by distance ascending, limited to req.Limit.
func (g *RedisGeo) NearbyDrivers(ctx context.Context, req models.NearbyRequest) ([]models.NearbyDriver, error) {
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	locations, err := g.rdb.GeoSearchLocation(ctx, geoKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  req.Lng,
			Latitude:   req.Lat,
			Radius:     req.RadiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      limit,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("geo search: %w", err)
	}

	// Filter to only online drivers using a pipeline of SISMEMBER calls
	drivers := make([]models.NearbyDriver, 0, len(locations))
	for _, loc := range locations {
		online, _ := g.IsOnline(ctx, loc.Name)
		if !online {
			continue
		}
		drivers = append(drivers, models.NearbyDriver{
			DriverID:   loc.Name,
			Lat:        loc.Latitude,
			Lng:        loc.Longitude,
			DistanceKm: loc.Dist,
			IsOnline:   true,
		})
	}
	return drivers, nil
}

func (g *RedisGeo) GetDriverLocation(ctx context.Context, driverID string) (*models.DriverLocation, error) {
	// GeoPos returns [lon, lat] for a member
	positions, err := g.rdb.GeoPos(ctx, geoKey, driverID).Result()
	if err != nil || len(positions) == 0 || positions[0] == nil {
		return nil, fmt.Errorf("driver %s not found", driverID)
	}

	metaKey := fmt.Sprintf(metaKeyFmt, driverID)
	meta, _ := g.rdb.HGetAll(ctx, metaKey).Result()

	loc := &models.DriverLocation{
		DriverID: driverID,
		Lng:      positions[0].Longitude,
		Lat:      positions[0].Latitude,
	}
	if v, ok := meta["updated_at"]; ok {
		loc.UpdatedAt, _ = time.Parse(time.RFC3339Nano, v)
	}
	return loc, nil
}

func (g *RedisGeo) SetOnline(ctx context.Context, driverID string) error {
	// SADD + separate per-driver expiry key pattern
	_, err := g.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.SAdd(ctx, onlineKey, driverID)
		// Individual expiry tracked via a separate key
		p.Set(ctx, fmt.Sprintf("ridego:driver:%s:online", driverID), 1, ttlOnline)
		return nil
	})
	return err
}

func (g *RedisGeo) SetOffline(ctx context.Context, driverID string) error {
	_, err := g.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.SRem(ctx, onlineKey, driverID)
		p.Del(ctx, fmt.Sprintf("ridego:driver:%s:online", driverID))
		return nil
	})
	return err
}

func (g *RedisGeo) IsOnline(ctx context.Context, driverID string) (bool, error) {
	// Check the per-driver TTL key (not the SADD) — this auto-expires
	n, err := g.rdb.Exists(ctx, fmt.Sprintf("ridego:driver:%s:online", driverID)).Result()
	return n > 0, err
}
