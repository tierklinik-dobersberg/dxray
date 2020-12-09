package models

import (
	"encoding/xml"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html/charset"
)

type (
	// ImageList is stored inside study.xml. It contains patient data
	ImageList struct {
		XMLName xml.Name `xml:"Imagelist" json:"-"`
		Patient Patient  `xml:"Patient"`
	}

	// Patient describes a patient
	Patient struct {
		XMLName xml.Name `xml:"Patient" json:"-"`
		Name    string   `xml:"Name"`
		ID      string   `xml:"ID"`
		Birth   string   `xml:"Birth"`
		Sex     string   `xml:"Sex"`
		Visit   Visit    `xml:"Visit"`
	}

	// Visit is a visit of a patient and contains a study
	Visit struct {
		XMLName xml.Name `xml:"Visit" json:"-"`
		Study   Study    `xml:"Study"`
	}

	// Study represents a study done during the Visit of a Patient
	Study struct {
		XMLName     xml.Name `xml:"Study" json:"-"`
		UID         string   `xml:"UID"`
		Date        string   `xml:"Date"`
		Description string   `xml:"Description"`
		Series      []Series `xml:"Series"`
	}

	// Series is a series of medical imaging pictures taken during a study
	Series struct {
		XMLName     xml.Name   `xml:"Series" json:"-"`
		UID         string     `xml:"UID"`
		Number      int        `xml:"Number"`
		Description string     `xml:"Description"`
		Protocol    string     `xml:"Protocol"`
		Modality    string     `xml:"Modality"`
		Instances   []Instance `xml:"Instance"`
	}

	// Instance is a medical picture take during a series
	Instance struct {
		XMLName xml.Name     `xml:"Instance" json:"-"`
		UID     string       `xml:"UID"`
		Number  int          `xml:"Number"`
		Data    InstanceData `xml:"Data" json:"-"`
	}

	// InstanceData describes the path to the medical image of an Instance
	InstanceData struct {
		XMLName   xml.Name `xml:"Data" json:"-"`
		DICOMPath string   `xml:"DICOM" json:"-"`
	}
)

// OwnerName returns the name of the patient owner. DX-R stores that information
// concatinated with the animal name and race
func (p Patient) OwnerName() string {
	parts := strings.Split(p.Name, "^")
	if len(parts) != 2 {
		return p.Name
	}

	return parts[0]
}

// AnimalName tries to extract the name of the anima. DX-R stores that information
// concatinated with the owner name and race
func (p Patient) AnimalName() string {
	nameParts := strings.Split(p.Name, "^")
	if len(nameParts) != 2 {
		return "unknown"
	}

	parts := strings.Split(strings.TrimSpace(nameParts[1]), " ")
	return parts[0]
}

// AnimalRace tries to return the race of the animal. DX-R stores that information
// concatinated with the owner and animal name
func (p Patient) AnimalRace() string {
	nameParts := strings.Split(p.Name, "^")
	if len(nameParts) != 2 {
		return "unknown"
	}

	parts := strings.Split(strings.TrimSpace(nameParts[1]), " ")
	if len(parts) == 1 {
		return "unknown"
	}

	race := strings.Join(parts[1:], " ")
	return race
}

// FromReader reads a ImageList from r
func FromReader(r io.Reader) (*ImageList, error) {
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel

	var i ImageList

	if err := decoder.Decode(&i); err != nil {
		return nil, err
	}

	return &i, nil
}

// FromFile reads a ImageList from path
func FromFile(path string) (*ImageList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return FromReader(f)
}
