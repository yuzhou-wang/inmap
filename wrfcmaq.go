/*
Copyright © 2013 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"fmt"
	"time"
	"math"

	"github.com/ctessum/atmos/seinfeld"
	"github.com/ctessum/atmos/wesely1989"

	"github.com/ctessum/sparse"
)

// WRF variables currently used:
/* hc5,hc8,olt,oli,tol,xyl,csl,cvasoa1,cvasoa2,cvasoa3,cvasoa4,iso,api,sesq,lim,
cvbsoa1,cvbsoa2,cvbsoa3,cvbsoa4,asoa1i,asoa1j,asoa2i,asoa2j,asoa3i,asoa3j,asoa4i,
asoa4j,bsoa1i,bsoa1j,bsoa2i,bsoa2j,bsoa3i,bsoa3j,bsoa4i,bsoa4j,no,no2,no3ai,no3aj,
so2,sulf,so4ai,so4aj,nh3,nh4ai,nh4aj,PM2_5_DRY,U,V,W,PBLH,PH,PHB,HFX,UST,PBLH,T,
PB,P,ho,h2o2,LU_INDEX,QRAIN,CLDFRA,QCLOUD,ALT,SWDOWN,GLW */

const cmaqFormat = "2006-01-02"
// = "aVOC            bVOC            aSOA            bSOA            bOrgPartitioningaOrgPartitioningTotalPM25       gNH             gNO             gS              pNH             pNO             pS              NHPartitioning  NOPartitioning  SPartitioning   NO_NO2partitioni" ;
// aVOC bVOC aSOA bSOA pNO pS pNH totalPM25=TotalPM25 sox=gS nox=gNO nh3=gNH
// WRFCmaq is an InMAP preprocessor for WRF-Cmaq output.
type WRFCmaq struct {
//	aVOC, bVOC, aSOA, bSOA, nox, no, no2, pNO, sox, pS, nh3, pNH, totalPM25 map[string]float64
	aVOC, bVOC, aSOA, bSOA, nox, pNO, sox, pS, nh3, pNH, totalPM25 map[string]float64

	start, end time.Time

	cmaqOut string

	recordDelta, fileDelta time.Duration

	msgChan chan string
}

// NewWRFCmaq initializes a WRF-Cmaq preprocessor from the given
// configuration information.
// WRFOut is the location of WRF-Cmaq output files.
// [DATE] should be used as a wild card for the simulation date.
// startDate and endDate are the dates of the beginning and end of the
// simulation, respectively, in the format "YYYYMMDD".
// If msgChan is not nil, status messages will be sent to it.
func NewWRFCmaq(WRFOut, startDate, endDate string, msgChan chan string) (*WRFCmaq, error) {
	w := WRFCmaq{
                // totalPM25 is total mass of PM2.5  [μg/m3].
                totalPM25: map[string]float64{"TotalPM25": 1.},
                sox: map[string]float64{"gS": 1.},
                nox: map[string]float64{"gNO": 1.},
                nh3: map[string]float64{"gNH": 1.},
                aVOC: map[string]float64{"aVOC": 1.},
                bVOC: map[string]float64{"bVOC": 1.},
                aSOA: map[string]float64{"aSOA": 1.},
                bSOA: map[string]float64{"bSOA": 1.},
                pNO: map[string]float64{"pNO": 1.},
                pS: map[string]float64{"pS": 1.},
                pNH: map[string]float64{"pNH": 1.},
		cmaqOut:  WRFOut,
		msgChan: msgChan,
	}

	var err error
	w.start, err = time.Parse(inDateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Cmaq preprocessor start time: %v", err)
	}
	w.end, err = time.Parse(inDateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Cmaq preprocessor end time: %v", err)
	}

	w.recordDelta, err = time.ParseDuration("1h")
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Cmaq preprocessor recordDelta: %v", err)
	}
	w.fileDelta, err = time.ParseDuration("24h")
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Cmaq preprocessor fileDelta: %v", err)
	}
	return &w, nil
}


func (w *WRFCmaq) read(varName string) NextData {
	return nextDataNCF(w.cmaqOut, cmaqFormat, varName, w.start, w.end, w.recordDelta, w.fileDelta, readNCF, w.msgChan)
}

func (w *WRFCmaq) readGroup(varGroup map[string]float64) NextData {
	return nextDataGroupNCF(w.cmaqOut, cmaqFormat, varGroup, w.start, w.end, w.recordDelta, w.fileDelta, readNCF, w.msgChan)
}

// Nx helps fulfill the Preprocessor interface by returning
// the number of grid cells in the West-East direction.
func (w *WRFCmaq) Nx() (int, error) {
	f, ff, err := ncfFromTemplate(w.cmaqOut, cmaqFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("nx: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[3], nil
}

// Ny helps fulfill the Preprocessor interface by returning
// the number of grid cells in the South-North direction.
func (w *WRFCmaq) Ny() (int, error) {
	f, ff, err := ncfFromTemplate(w.cmaqOut, cmaqFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("ny: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[2], nil
}

// Nz helps fulfill the Preprocessor interface by returning
// the number of grid cells in the below-above direction.
func (w *WRFCmaq) Nz() (int, error) {
	f, ff, err := ncfFromTemplate(w.cmaqOut, cmaqFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("nz: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[1], nil
}

// PBLH helps fulfill the Preprocessor interface by returning
// planetary boundary layer height [m].
func (w *WRFCmaq) PBLH() NextData { return w.read("PBLH") }

// Height helps fulfill the Preprocessor interface by returning
// layer heights above ground level calculated based on geopotential height.
// For more information, refer to
// http://www.openwfm.org/wiki/How_to_interpret_WRF_variables.
func (w *WRFCmaq) Height() NextData {
	// ph is perturbation geopotential height [m2/s].
	phFunc := w.read("PH")
	// phb is baseline geopotential height [m2/s].
	phbFunc := w.read("PHB")
	return func() (*sparse.DenseArray, error) {
		ph, err := phFunc()
		if err != nil {
			return nil, err
		}
		phb, err := phbFunc()
		if err != nil {
			return nil, err
		}
		return geopotentialToHeight(ph, phb), nil
	}
}


// ALT helps fulfill the Preprocessor interface by returning
// inverse air density [m3/kg].
func (w *WRFCmaq) ALT() NextData { return w.read("ALT") }

// U helps fulfill the Preprocessor interface by returning
// West-East wind speed [m/s].
func (w *WRFCmaq) U() NextData { return w.read("U") }

// V helps fulfill the Preprocessor interface by returning
// South-North wind speed [m/s].
func (w *WRFCmaq) V() NextData { return w.read("V") }

// W helps fulfill the Preprocessor interface by returning
// below-above wind speed [m/s].
func (w *WRFCmaq) W() NextData { return w.read("W") }

// AVOC helps fulfill the Preprocessor interface.
func (w *WRFCmaq) AVOC() NextData { return w.readGroup(w.aVOC) }

// BVOC helps fulfill the Preprocessor interface.
func (w *WRFCmaq) BVOC() NextData { return w.readGroup(w.bVOC) }

// NOx helps fulfill the Preprocessor interface.
func (w *WRFCmaq) NOx() NextData { return w.readGroup(w.nox) }

// SOx helps fulfill the Preprocessor interface.
func (w *WRFCmaq) SOx() NextData { return w.readGroup(w.sox) }

// NH3 helps fulfill the Preprocessor interface.
func (w *WRFCmaq) NH3() NextData { return w.readGroup(w.nh3) }

// ASOA helps fulfill the Preprocessor interface.
func (w *WRFCmaq) ASOA() NextData { return w.readGroup(w.aSOA) }

// BSOA helps fulfill the Preprocessor interface.
func (w *WRFCmaq) BSOA() NextData { return w.readGroup(w.bSOA) }

// PNO helps fulfill the Preprocessor interface.
func (w *WRFCmaq) PNO() NextData { return w.readGroup(w.pNO) }

// PS helps fulfill the Preprocessor interface.
func (w *WRFCmaq) PS() NextData { return w.readGroup(w.pS) }

// PNH helps fulfill the Preprocessor interface.
func (w *WRFCmaq) PNH() NextData { return w.readGroup(w.pNH) }

// TotalPM25 helps fulfill the Preprocessor interface.
func (w *WRFCmaq) TotalPM25() NextData { return w.readGroup(w.totalPM25) }

// SurfaceHeatFlux helps fulfill the Preprocessor interface
// by returning heat flux at the surface [W/m2].
func (w *WRFCmaq) SurfaceHeatFlux() NextData { return w.read("HFX") }

// UStar helps fulfill the Preprocessor interface
// by returning friction velocity [m/s].
func (w *WRFCmaq) UStar() NextData { return w.read("UST") }

// T helps fulfill the Preprocessor interface by
// returning temperature [K].
func (w *WRFCmaq) T() NextData {
	thetaFunc := w.read("T") // perturbation potential temperature [K]
	pFunc := w.P()           // Pressure [Pa]
	return cmaqTemperatureConvert(thetaFunc, pFunc)
}

func cmaqTemperatureConvert(thetaFunc, pFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		thetaPerturb, err := thetaFunc() // perturbation potential temperature [K]
		if err != nil {
			return nil, err
		}
		p, err := pFunc() // Pressure [Pa]
		if err != nil {
			return nil, err
		}

		T := sparse.ZerosDense(thetaPerturb.Shape...)
		for i, tp := range thetaPerturb.Elements {
			T.Elements[i] = thetaPerturbToTemperature(tp, p.Elements[i])
		}
		return T, nil
	}
}


// P helps fulfill the Preprocessor interface
// by returning pressure [Pa].
func (w *WRFCmaq) P() NextData {
	pbFunc := w.read("PB") // baseline pressure [Pa]
	pFunc := w.read("P")   // perturbation pressure [Pa]
	return cmaqPressureConvert(pFunc, pbFunc)
}

func cmaqPressureConvert(pFunc, pbFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		pb, err := pbFunc() // baseline pressure [Pa]
		if err != nil {
			return nil, err
		}
		p, err := pFunc() // perturbation pressure [Pa]
		if err != nil {
			return nil, err
		}
		P := pb.Copy()
		P.AddDense(p)
		return P, nil
	}
}

// HO helps fulfill the Preprocessor interface
// by returning hydroxyl radical concentration [ppmv].
func (w *WRFCmaq) HO() NextData { return w.read("oh") }

// H2O2 helps fulfill the Preprocessor interface
// by returning hydrogen peroxide concentration [ppmv].
func (w *WRFCmaq) H2O2() NextData { return w.read("h2o2") }

// SeinfeldLandUse helps fulfill the Preprocessor interface
// by returning land use categories as
// specified in github.com/ctessum/atmos/seinfeld.
func (w *WRFCmaq) SeinfeldLandUse() NextData {
	luFunc := w.read("LU_INDEX") // USGS land use index
	return cmaqSeinfeldLandUse(luFunc)
}

func cmaqSeinfeldLandUse(luFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		lu, err := luFunc() // USGS land use index
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(lu.Shape...)
		for j := 0; j < lu.Shape[0]; j++ {
			for i := 0; i < lu.Shape[1]; i++ {
				o.Set(float64(NLCDseinfeld[f2i(lu.Get(j, i)) - 1]), j, i)
			}
		}
		return o, nil
	}
}

// NLCDseinfeld lookup table to go from USGS land classes to land classes for
// particle dry deposition.
var NLCDseinfeld = []seinfeld.LandUseCategory{
	seinfeld.Evergreen, //'Evergreen Needleleaf Forest'
        seinfeld.Deciduous, //'Evergreen Broadleaf Forest'
        seinfeld.Evergreen, //'Deciduous Needleleaf Forest'
        seinfeld.Deciduous, //'Deciduous Broadleaf Forest'
        seinfeld.Deciduous, //'Mixed Forest'
        seinfeld.Shrubs,    //'Closed Shrubland'
        seinfeld.Shrubs,    //'Open Shrubland'
        seinfeld.Shrubs,    //'Woody Savanna'
        seinfeld.Grass,     //'Savanna'
        seinfeld.Grass,     //'Grassland'
        seinfeld.Grass,     //'Permanent Wetland'
        seinfeld.Grass,     //'Cropland'
        seinfeld.Desert,    //'Urban and Built-Up'
        seinfeld.Grass,     //'Cropland / Natural Veg. Mosaic'
        seinfeld.Desert,    //'Permanent Snow'
        seinfeld.Desert,    //'Barren / Sparsely Vegetated'
        seinfeld.Desert,    //'IGBP Water'
        seinfeld.Desert,    //'Unclassified'
        seinfeld.Desert,    //'Fill Value'
        seinfeld.Desert,    //'Unclassified'
        seinfeld.Desert,    //'Open Water'
        seinfeld.Desert,    //'Perennial Ice/Snow'
        seinfeld.Desert,    //'Developed Open Space'
        seinfeld.Desert,    //'Developed Low Intensity'
        seinfeld.Desert,    //'Developed Medium Intensity'
        seinfeld.Desert,    //'Developed High Intensity'
        seinfeld.Desert,    //'Barren Land'
        seinfeld.Deciduous, //'Deciduous Forest'
        seinfeld.Evergreen, //'Evergreen Forest'
        seinfeld.Deciduous, //'Mixed Forest'
        seinfeld.Shrubs,    //'Dwarf Scrub'
        seinfeld.Shrubs,    //'Shrub/Scrub'
        seinfeld.Grass,     //'Grassland/Herbaceous'
        seinfeld.Grass,     //'Sedge/Herbaceous'
        seinfeld.Desert,    //'Lichens'
        seinfeld.Desert,    //'Moss'
        seinfeld.Grass,     //'Pasture/Hay'
        seinfeld.Grass,     //'Cultivated Crops'
        seinfeld.Deciduous, //'Woody Wetland'
        seinfeld.Grass,     //'Emergent Herbaceous Wetland'
}

// thetaPerturbToTemperature converts perburbation potential temperature
// to ambient temperature for the given pressure (p [Pa]).
func thetaPerturbToTemperature(thetaPerturb, p float64) float64 {
	const (
		po    = 101300. // Pa, reference pressure
		kappa = 0.2854  // related to von karman's constant
	)
	pressureCorrection := math.Pow(p/po, kappa)
	// potential temperature, K
	θ := thetaPerturb + 300.
	// Ambient temperature, K
	return θ * pressureCorrection
}

func geopotentialToHeight(ph, phb *sparse.DenseArray) *sparse.DenseArray {
	layerHeights := sparse.ZerosDense(ph.Shape...)
	for k := 0; k < ph.Shape[0]; k++ {
		for j := 0; j < ph.Shape[1]; j++ {
			for i := 0; i < ph.Shape[2]; i++ {
				h := (ph.Get(k, j, i) + phb.Get(k, j, i) -
					ph.Get(0, j, i) - phb.Get(0, j, i)) / g // m
				layerHeights.Set(h, k, j, i)
			}
		}
	}
	return layerHeights
}


// WeselyLandUse helps fulfill the Preprocessor interface
// by returning land use categories as
// specified in github.com/ctessum/atmos/wesely1989.
func (w *WRFCmaq) WeselyLandUse() NextData {
	luFunc := w.read("LU_INDEX") // NLCD land use index
	return cmaqWeselyLandUse(luFunc)
}

func cmaqWeselyLandUse(luFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		lu, err := luFunc() // NLCD land use index
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(lu.Shape...)
		for j := 0; j < lu.Shape[0]; j++ {
			for i := 0; i < lu.Shape[1]; i++ {
				o.Set(float64(NLCDwesely[f2i(lu.Get(j, i)) - 1]), j, i)
			}
		}
		return o, nil
	}
}

// NLCDwesely lookup table to go from NLCD land classes to land classes for
// gas dry deposition.
var NLCDwesely = []wesely1989.LandUseCategory{
	wesely1989.Coniferous,   //'Evergreen Needleleaf Forest'
        wesely1989.Deciduous,    //'Evergreen Broadleaf Forest'
        wesely1989.Coniferous,   //'Deciduous Needleleaf Forest'
        wesely1989.Deciduous,    //'Deciduous Broadleaf Forest'
        wesely1989.MixedForest,  //'Mixed Forest'
        wesely1989.RockyShrubs,  //'Closed Shrubland'
        wesely1989.RockyShrubs,  //'Open Shrubland'
        wesely1989.RockyShrubs,  //'Woody Savanna'
        wesely1989.Range,        //'Savanna'
        wesely1989.Range,        //'Grassland'
        wesely1989.Wetland,      //'Permanent Wetland'
        wesely1989.RangeAg,      //'Cropland'
        wesely1989.Urban,        //'Urban and Built-Up'
        wesely1989.RangeAg,      //'Cropland / Natural Veg. Mosaic'
        wesely1989.Barren,       //'Permanent Snow'
        wesely1989.Barren,       //'Barren / Sparsely Vegetated'
        wesely1989.Water,        //'IGBP Water'
        wesely1989.Barren,       //'Unclassified'
        wesely1989.Barren,       //'Fill Value'
        wesely1989.Barren,       //'Unclassified'
        wesely1989.Water,        //'Open Water'
        wesely1989.Barren,       //'Perennial Ice/Snow'
        wesely1989.Urban,        //'Developed Open Space'
        wesely1989.Urban,        //'Developed Low Intensity'
        wesely1989.Urban,        //'Developed Medium Intensity'
        wesely1989.Urban,        //'Developed High Intensity'
        wesely1989.Barren,       //'Barren Land'
        wesely1989.Deciduous,    //'Deciduous Forest'
        wesely1989.Coniferous,   //'Evergreen Forest'
        wesely1989.MixedForest,  //'Mixed Forest'
        wesely1989.RockyShrubs,  //'Dwarf Scrub'
        wesely1989.RockyShrubs,  //'Shrub/Scrub'
        wesely1989.Range,        //'Grassland/Herbaceous'
        wesely1989.Range,        //'Sedge/Herbaceous'
        wesely1989.Barren,       //'Lichens'
        wesely1989.Barren,       //'Moss'
        wesely1989.RangeAg,      //'Pasture/Hay'
        wesely1989.RangeAg,      //'Cultivated Crops'
        wesely1989.Wetland,      //'Woody Wetland'
        wesely1989.Wetland,      //'Emergent Herbaceous Wetland'
}


// Z0 helps fulfill the Preprocessor interface by
// returning roughness length.
func (w *WRFCmaq) Z0() NextData {
	LUIndexFunc := w.read("LU_INDEX") //NLCD land use index
	return cmaqZ0(LUIndexFunc)
}

// NLCDz0 holds Mean Roughness lengths for NLCD land classes ([m]), from WRF file
// VEGPARM.TBL.
var NLCDz0 = []float64{.50, .50, .50, .50, .35, .03, .035, .03, .15, .11,
        .30, .10, .50, .095, .001, .01, .0001, 999., 999., 999.,
        .0001, .001, .50, .70, 1.5, 2.0, .01, .50, .50, .35,
        .025, .03, .11, .20, .01, .01, .10, .06, .40, .20}

func cmaqZ0(LUIndexFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		luIndex, err := LUIndexFunc()
		if err != nil {
			return nil, err
		}
		zo := sparse.ZerosDense(luIndex.Shape...)
		for i, lu := range luIndex.Elements {
			zo.Elements[i] = NLCDz0[f2i(lu) - 1] // roughness length [m]
		}
		return zo, nil
	}
}

// QRain helps fulfill the Preprocessor interface by
// returning rain mass fraction.
func (w *WRFCmaq) QRain() NextData { return w.read("QRAIN") }

// CloudFrac helps fulfill the Preprocessor interface
// by returning the fraction of each grid cell filled
// with clouds [volume/volume].
func (w *WRFCmaq) CloudFrac() NextData { return w.read("CLDFRA") }

// QCloud helps fulfill the Preprocessor interface by returning
// the mass fraction of cloud water in each grid cell [mass/mass].
func (w *WRFCmaq) QCloud() NextData { return w.read("QCLOUD") }

// RadiationDown helps fulfill the Preprocessor interface by returning
// total downwelling radiation at ground level [W/m2].
func (w *WRFCmaq) RadiationDown() NextData {
	swDownFunc := w.read("SWDOWN") // downwelling short wave radiation at ground level [W/m2]
	glwFunc := w.read("GLW")       // downwelling long wave radiation at ground level [W/m2]
	return cmaqRadiationDown(swDownFunc, glwFunc)
}

func cmaqRadiationDown(swDownFunc, glwFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		swDown, err := swDownFunc() // downwelling short wave radiation at ground level [W/m2]
		if err != nil {
			return nil, err
		}
		glw, err := glwFunc() // downwelling long wave radiation at ground level [W/m2]
		if err != nil {
			return nil, err
		}
		rad := swDown.Copy()
		rad.AddDense(glw)
		return rad, nil
	}
}

// SWDown helps fulfill the Preprocessor interface by returning
// downwelling short wave radiation at ground level [W/m2].
func (w *WRFCmaq) SWDown() NextData { return w.read("SWDOWN") }

// GLW helps fulfill the Preprocessor interface by returning
// downwelling long wave radiation at ground level [W/m2].
func (w *WRFCmaq) GLW() NextData { return w.read("GLW") }
