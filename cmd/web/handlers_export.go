package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vsimakhin/web-logbook/internal/csvexport"
	"github.com/vsimakhin/web-logbook/internal/models"
	"github.com/vsimakhin/web-logbook/internal/pdfexport"
	"github.com/vsimakhin/web-logbook/internal/xlsexport"
)

const exportA4 = "A4"
const exportA5 = "A5"
const exportCSV = "csv"
const exportXLS = "xls"

// HandlerExport is a handler for /export page
func (app *application) HandlerExport(w http.ResponseWriter, r *http.Request) {
	if app.config.env == "dev" {
		app.infoLog.Println(APIExport)
	}

	settings, err := app.db.GetSettings()
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := make(map[string]interface{})
	data["settings"] = settings

	partials := []string{"common-js", "export-js", "export-a4", "export-a5", "export-xls", "export-csv"}
	if err := app.renderTemplate(w, r, "export", &templateData{Data: data}, partials...); err != nil {
		app.errorLog.Println(err)
	}
}

// HandlerExportLogbook executes the pdf export and returns pdf file
func (app *application) HandlerExportLogbook(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")

	if app.config.env == "dev" {
		app.infoLog.Println(strings.ReplaceAll(APIExportFormat, "{format}", format))
	}

	flightRecords, err := app.db.GetFlightRecords()
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings, err := app.db.GetSettings()
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if format == exportA4 || format == exportA5 {
		// export to PDF

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=logbook.pdf")

		var logbook pdfexport.Logbook
		logbook.OwnerName = settings.OwnerName
		logbook.Signature = settings.SignatureText

		if format == exportA4 {
			logbook.PageBreaks = strings.Split(settings.ExportA4.PageBreaks, ",")
			logbook.Export = settings.ExportA4

			err = logbook.ExportA4(flightRecords, w)

		} else if format == exportA5 {
			logbook.PageBreaks = strings.Split(settings.ExportA5.PageBreaks, ",")
			logbook.Export = settings.ExportA5

			err = logbook.ExportA5(flightRecords, w)
		}

	} else if format == exportCSV {
		// export to CSV format
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=logbook.csv")

		var e csvexport.ExportCSV
		e.ExportCSV = settings.ExportCSV
		err = e.Export(flightRecords, w)

	} else if format == exportXLS {
		// export to XLS format

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=logbook.xlsx")

		var xls xlsexport.ExportXLS
		xls.ExportXLS = settings.ExportXLS

		err = xls.Export(flightRecords, w)

	} else {
		err = errors.New("unknown export format")

	}

	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandlerExportSettingsSave serves the POST request for export settings update
func (app *application) HandlerExportSettingsSave(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")

	if app.config.env == "dev" {
		app.infoLog.Println(strings.ReplaceAll(APIExportFormat, "{format}", format))
	}

	currsettings, err := app.db.GetSettings()
	if err != nil {
		app.errorLog.Println(fmt.Errorf("cannot get the export settings - %s", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var settings models.Settings
	var response models.JSONResponse

	err = json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		app.errorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// rewrite only export settings since the others are set from /settings page
	if format == exportA4 {
		currsettings.ExportA4 = settings.ExportA4

	} else if format == exportA5 {
		currsettings.ExportA5 = settings.ExportA5

	} else if format == exportCSV {
		currsettings.ExportCSV = settings.ExportCSV

	} else if format == exportXLS {
		currsettings.ExportXLS = settings.ExportXLS

	}

	err = app.db.UpdateSettings(currsettings)
	if err != nil {
		app.errorLog.Println(err)
		response.OK = false
		response.Message = err.Error()
	} else {
		response.OK = true
		response.Message = "Export settings have been updated"

		if app.config.env == "dev" {
			app.infoLog.Println("export settings updated")
		}
	}

	err = app.writeJSON(w, http.StatusOK, response)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
}
