package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AWAKENS-dev/awtk/lib"
	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"
	"github.com/urfave/cli"
)

func doRunServer(c *cli.Context) {
	awtk.InitDatabase()

	addr := c.String("addr")
	if addr == "" {
		addr = "localhost:1323"
	}

	log.WithFields(log.Fields{"addr": addr, "awtk_version": Version}).Info("Running awtk server")

	e := echo.New()
	e.Use(middleware.Logger())

	e.POST("/v1/genomes", postGenomes)
	e.GET("/v1/genomes", getGenomesList)
	e.GET("/v1/genomes/:genome_id", getGenomes)
	e.GET("/v1/genomes/:genome_id/genotypes", getGenotypes)
	e.GET("/v1/evidences/:evidence_id", getEvidences)

	e.Run(standard.New(addr))
}

// postGenomes creates genomes records by filePath
// $ curl -X POST --data "filePath=test/data/test.vcf41.vcf.gz" "http://localhost:1323/v1/genomes"
func postGenomes(c echo.Context) error {
	g := new(awtk.Genome)
	if err := c.Bind(g); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	genomes, err := awtk.CreateGenomes(g.FilePath)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusCreated, genomes)
}

// getGenomes returns all genome records
func getGenomesList(c echo.Context) error {
	genomes, err := awtk.GetGenomes()
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, genomes)
}

// getGenomes returns genome record by id
func getGenomes(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("genome_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	genome, err := awtk.GetGenome(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, genome)
}

// getGenotypes returns genotypes records of genome by locations
// $ curl "localhost:1323/v1/genomes/1/genotypes?locations=1:1,1:2,1:3"
// $ curl "localhost:1323/v1/genomes/1/genotypes?range=1:1-100"
func getGenotypes(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("genome_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	genome, err := awtk.GetGenome(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	locationsParam := c.QueryParam("locations")
	rangeParam := c.QueryParam("range")

	if locationsParam != "" && rangeParam != "" {
		err = &awtk.GenomeError{fmt.Sprintf("%s", "Invalid query param. Both locations and range params found.")}
		return c.JSON(http.StatusBadRequest, err)
	}

	if locationsParam == "" && rangeParam == "" {
		err = &awtk.GenomeError{fmt.Sprintf("%s", "No valid query params found.")}
		return c.JSON(http.StatusBadRequest, err)
	}

	var locs []awtk.Location

	if locationsParam != "" {
		// e.g. ?locations=1:1,1:2,1:3
		queries := strings.Split(locationsParam, ",")
		for i := range queries {
			q := strings.Split(queries[i], ":")
			if len(q) != 2 {
				err = &awtk.GenomeError{fmt.Sprintf("%s", "Invalid locations")}
				return c.JSON(http.StatusBadRequest, err)
			}

			pos, err := strconv.Atoi(q[1])
			if err != nil {
				return c.JSON(http.StatusBadRequest, err)
			}
			loc := awtk.NewLocation(q[0], pos-1, pos) // 1-based to 0-based
			locs = append(locs, loc)
		}
	} else if rangeParam != "" {
		// e.g. ?range=1:1-100
		query := rangeParam
		q := strings.Split(query, ":")
		if len(q) != 2 {
			err = &awtk.GenomeError{fmt.Sprintf("%s", "Invalid locations")}
			return c.JSON(http.StatusBadRequest, err)
		}
		r := strings.Split(q[1], "-")
		if len(r) != 2 {
			err = &awtk.GenomeError{fmt.Sprintf("%s", "Invalid locations")}
			return c.JSON(http.StatusBadRequest, err)
		}

		start, err := strconv.Atoi(r[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		end, err := strconv.Atoi(r[1])
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		loc := awtk.NewLocation(q[0], start-1, end) // 1-based to 0-based
		locs = append(locs, loc)
	}

	genotypes, err := awtk.QueryGenotypes(genome.FilePath, genome.SampleIndex, locs)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	// &fmt=seq
	fmtParam := c.QueryParam("fmt")
	if rangeParam != "" && fmtParam == "seq" {
		seq, err := awtk.Genotypes2Sequence(genotypes, locs)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		return c.JSON(http.StatusOK, seq)
	}

	return c.JSON(http.StatusOK, genotypes)
}

//
func getEvidences(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("evidence_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	evidence, err := awtk.GetEvidence(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.String(http.StatusOK, string(evidence))
}
