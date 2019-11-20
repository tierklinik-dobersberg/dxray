package ohif

import (
	"strconv"

	"github.com/apex/log"
	"github.com/grailbio/go-dicom"
	"github.com/grailbio/go-dicom/dicomtag"
	"github.com/tierklinik-dobersberg/dxray/pkg/dxr/fsdb"
)

type (
	// StudyJSON is describes the response structure used by OHIF standalone
	// viewer
	StudyJSON struct {
		UID              string   `json:"studyInstanceUid,omitempty"`
		Date             string   `json:"studyDate,omitempty"`
		Time             string   `json:"studyTime,omitempty"`
		PatientName      string   `json:"patientName,omitempty"`
		PatientAge       string   `json:"patientAge,omitempty"`
		PatientBirthDate string   `json:"patientBirthDate,omitempty"`
		PatientID        string   `json:"patientId,omitempty"`
		PatientSex       string   `json:"patientSex,omitempty"`
		Series           []Series `json:"seriesList,omitempty"`
	}

	// Series describes a series of medical images
	Series struct {
		Description string     `json:"seriesDescription,omitempty"`
		UID         string     `json:"seriesInstanceUid,omitempty"`
		BodyPart    string     `json:"seriesBodyPart,omitempty"`
		Number      string     `json:"seriesNumber,omitempty"`
		Date        string     `json:"seriesDate,omitempty"`
		Time        string     `json:"seriesTime,omitempty"`
		Modality    string     `json:"seriesModality,omitempty"`
		Instances   []Instance `json:"instances,omitempty"`
	}

	Instance struct {
		Number string `json:"instanceNumber"`
		UID    string `json:"sopInstanceUid"`
		URL    string `json:"url"`

		*InstanceTags
	}

	InstanceTags struct {
		Columns                   uint16 `json:"columns"`
		Rows                      uint16 `json:"rows"`
		PhotometricInterpretation string `json:"photometricInterpretation"`
		BitAllocated              uint16 `json:"bitAllocated"`
		BitsStored                uint16 `json:"bitsStored"`
		PixelRepresentation       uint16 `json:"pixelRepresentation"`
		SamplesPerPixel           uint16 `json:"samplesPerPixel"`
		HighBit                   uint16 `json:"highBit"`
		RescaleSlope              string `json:"rescaleSlope"`
		RescaleIntercept          string `json:"rescaleIntercept"`
		ImageType                 string `json:"imageType"`
	}
)

// JSONFromDXR returns the JSON format required by OHIF viewer
// from the study.xml file stored by DX-R
func JSONFromDXR(study fsdb.Study, instanceURL func(string, string, string) string, withTags bool) (*StudyJSON, error) {
	if err := study.Load(); err != nil {
		return nil, err
	}

	xml, _ := study.Model()
	s := xml.Patient.Visit.Study
	model := &StudyJSON{
		UID:              s.UID,
		Date:             s.Date,
		PatientName:      xml.Patient.Name,
		PatientBirthDate: xml.Patient.Birth,
		PatientID:        xml.Patient.ID,
		PatientSex:       xml.Patient.Sex,
	}

	for _, series := range s.Series {
		sm := Series{
			Description: series.Description,
			UID:         series.UID,
			Number:      strconv.Itoa(series.Number),
			Modality:    series.Modality,
		}

		for _, instance := range series.Instances {
			im := Instance{
				UID:    instance.UID,
				Number: strconv.Itoa(instance.Number),
				URL:    instanceURL(s.UID, series.UID, instance.UID),
			}

			if withTags {
				path := study.RealPath(instance.Data.DICOMPath)
				im.InstanceTags = &InstanceTags{}
				if err := setDCMTags(path, im.InstanceTags); err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
						"path":  path,
					}).Errorf("failed to set tags from DCM file")
				}
			}

			sm.Instances = append(sm.Instances, im)
		}

		model.Series = append(model.Series, sm)
	}

	return model, nil
}

func setDCMTags(path string, i *InstanceTags) error {
	ds, err := dicom.ReadDataSetFromFile(path, dicom.ReadOptions{DropPixelData: true})
	if err != nil {
		return err
	}

	// TODO(ppacher): check if we are missing something
	i.Columns = uint16Tag(ds, dicomtag.Columns)
	i.Rows = uint16Tag(ds, dicomtag.Rows)
	i.PhotometricInterpretation = stringTag(ds, dicomtag.PhotometricInterpretation)
	i.BitAllocated = uint16Tag(ds, dicomtag.BitsAllocated)
	i.BitsStored = uint16Tag(ds, dicomtag.BitsStored)
	i.PixelRepresentation = uint16Tag(ds, dicomtag.PixelRepresentation)
	i.SamplesPerPixel = uint16Tag(ds, dicomtag.SamplesPerPixel)
	i.HighBit = uint16Tag(ds, dicomtag.HighBit)
	i.RescaleSlope = stringTag(ds, dicomtag.RescaleSlope)
	i.RescaleIntercept = stringTag(ds, dicomtag.RescaleIntercept)
	i.ImageType = stringTag(ds, dicomtag.ImageType)

	return nil
}

func stringTag(ds *dicom.DataSet, tag dicomtag.Tag) string {
	el, err := ds.FindElementByTag(tag)
	if err != nil {
		return ""
	}
	s, err := el.GetString()
	if err != nil {
		return ""
	}

	return s
}

func uint16Tag(ds *dicom.DataSet, tag dicomtag.Tag) uint16 {
	el, err := ds.FindElementByTag(tag)
	if err != nil {
		return 0
	}
	i, err := el.GetUInt16()
	if err != nil {
		return 0
	}

	return i
}
