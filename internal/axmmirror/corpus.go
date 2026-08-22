package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	pathpkg "path"
	"slices"
	"sort"
	"strings"
)

const (
	CorpusEvidenceLensSchema = "axm.waldo-witness.corpus-evidence-lens/v0.1"
	CorpusEvidenceStateReady = "READY"

	waldoTextRecordSchema = 1
	waldoTextWriterRecipe = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v5-bom"
	waldoFormerTextRecipe = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v4"
)

// EvidenceMeasures is WALDO's exact additive shard, document, token, and byte
// accounting. It is copied into receipts rather than recomputed by a model.
type EvidenceMeasures struct {
	Shards int64 `json:"shards"`
	Docs   int64 `json:"docs"`
	Tokens int64 `json:"tokens"`
	Bytes  int64 `json:"bytes"`
}

// ModalityEvidence preserves WALDO's per-modality accounting without treating
// samples from different modalities as mutually exclusive documents.
type ModalityEvidence struct {
	Samples      int64 `json:"samples,omitempty"`
	Items        int64 `json:"items,omitempty"`
	Tokens       int64 `json:"tokens,omitempty"`
	DurationMS   int64 `json:"duration_ms,omitempty"`
	ContentBytes int64 `json:"content_bytes,omitempty"`
}

type CorpusIndexEvidence struct {
	Remote string `json:"remote,omitempty"`
	Commit string `json:"commit,omitempty"`
	Dirty  bool   `json:"dirty"`
	State  string `json:"state"`
}

type CorpusLicensePolicy struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type SourceLicenseAssertion struct {
	Normalized  string `json:"normalized,omitempty"`
	Declaration string `json:"declaration,omitempty"`
	EvidenceURL string `json:"evidence_url,omitempty"`
	State       string `json:"state"`
}

type CorpusSourceEvidence struct {
	Name              string                 `json:"name"`
	Source            string                 `json:"source"`
	Version           string                 `json:"version,omitempty"`
	URL               string                 `json:"url"`
	SHA256            string                 `json:"sha256"`
	License           SourceLicenseAssertion `json:"license"`
	SourceFileCount   int                    `json:"source_file_count"`
	SourceFilesSHA256 string                 `json:"source_files_sha256,omitempty"`
}

type CorpusManifestEvidence struct {
	Path              string                 `json:"path"`
	SHA256            string                 `json:"sha256"`
	Name              string                 `json:"name"`
	LicenseAssertions []string               `json:"license_assertions"`
	Sources           []CorpusSourceEvidence `json:"sources"`
}

type CorpusLicenseEvidence struct {
	ID       string           `json:"id"`
	State    string           `json:"state"`
	Measures EvidenceMeasures `json:"measures"`
}

type CorpusAttestationEvidence struct {
	State         string `json:"state"`
	Embedded      int64  `json:"embedded"`
	ImplicitV4    int64  `json:"implicit_v4"`
	DeepValidated int64  `json:"deep_validated"`
	NotRecorded   int64  `json:"not_recorded"`
}

// CorpusEvidenceLens is a bounded, deterministic projection of a documented
// OpenWALDO corpus BOM. READY means that the receipt passed the structural and
// aggregate checks implemented here; the boundary notices state what it does
// not establish.
type CorpusEvidenceLens struct {
	Schema          string                      `json:"schema"`
	State           string                      `json:"state"`
	BOMSHA256       string                      `json:"bom_sha256"`
	DocumentSHA256  string                      `json:"document_sha256"`
	ReceiptSHA256   string                      `json:"receipt_sha256,omitempty"`
	Index           CorpusIndexEvidence         `json:"index"`
	Paths           []string                    `json:"paths"`
	LicensePolicy   CorpusLicensePolicy         `json:"license_policy"`
	Manifests       []CorpusManifestEvidence    `json:"manifests"`
	SourceSetSHA256 string                      `json:"source_set_sha256"`
	ShardSetSHA256  string                      `json:"shard_set_sha256"`
	Totals          EvidenceMeasures            `json:"totals"`
	Modalities      map[string]ModalityEvidence `json:"modalities,omitempty"`
	Licenses        []CorpusLicenseEvidence     `json:"licenses"`
	Attestation     CorpusAttestationEvidence   `json:"attestation"`
	Notices         []string                    `json:"notices"`
	Authority       Authority                   `json:"authority"`
}

// The wire types below mirror the documented schema-1 JSON order. They stay
// local to this fork: the organ consumes WALDO's durable artifact instead of
// importing WALDO implementation packages as an accidental API.
type waldoCorpusBOM struct {
	Kind         string                      `json:"kind"`
	Schema       int                         `json:"schema"`
	Subject      string                      `json:"subject"`
	Index        waldoCorpusIndex            `json:"index"`
	Paths        []string                    `json:"paths"`
	Policy       CorpusLicensePolicy         `json:"license_policy,omitempty"`
	Manifests    []waldoCorpusManifest       `json:"manifests"`
	SubManifests []waldoCorpusSubManifest    `json:"sub_manifests,omitempty"`
	Shards       []waldoCorpusShard          `json:"shards"`
	Totals       EvidenceMeasures            `json:"totals"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
	Licenses     map[string]EvidenceMeasures `json:"licenses"`
}

type waldoCorpusIndex struct {
	Remote string `json:"remote,omitempty"`
	Commit string `json:"commit,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
}

type waldoCorpusSubManifest struct {
	Manifest     string                      `json:"manifest"`
	ParentSHA256 string                      `json:"parent_sha256,omitempty"`
	URL          string                      `json:"url"`
	SHA256       string                      `json:"sha256"`
	Count        int64                       `json:"count"`
	Docs         int64                       `json:"docs"`
	Tokens       int64                       `json:"tokens"`
	Bytes        int64                       `json:"bytes"`
	EncodedBytes int64                       `json:"encoded_bytes"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
}

type waldoConversion struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Collector string `json:"collector,omitempty"`
	Profile   string `json:"profile"`
	Recipe    string `json:"recipe"`
	Tokenizer string `json:"tokenizer,omitempty"`
}

type waldoLicenseAssertion struct {
	Declaration string `json:"declaration,omitempty"`
	URL         string `json:"url,omitempty"`
}

type waldoSourceFile struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes,omitempty"`
	Format     string `json:"format,omitempty"`
	Adapter    string `json:"adapter,omitempty"`
	TextColumn string `json:"text_column,omitempty"`
}

type waldoContent struct {
	Types            []string `json:"types,omitempty"`
	Languages        []string `json:"languages,omitempty"`
	Geographies      []string `json:"geographies,omitempty"`
	Demographics     []string `json:"demographics,omitempty"`
	From             string   `json:"from,omitempty"`
	To               string   `json:"to,omitempty"`
	Selection        string   `json:"selection,omitempty"`
	PersonalData     string   `json:"personal_data,omitempty"`
	Copyrighted      string   `json:"copyrighted,omitempty"`
	MachineGenerated string   `json:"machine_generated,omitempty"`
}

type waldoCrawler struct {
	Name      string   `json:"name"`
	Purpose   string   `json:"purpose"`
	Behaviour string   `json:"behaviour"`
	Protocols []string `json:"protocols,omitempty"`
}

type waldoUserData struct {
	Service     string `json:"service"`
	Interaction string `json:"interaction"`
}

type waldoSyntheticData struct {
	Model       string `json:"model"`
	Version     string `json:"version,omitempty"`
	SummaryURL  string `json:"summary_url,omitempty"`
	Description string `json:"description,omitempty"`
}

type waldoDomainMeasure struct {
	Domain        string `json:"domain"`
	AcquiredBytes int64  `json:"acquired_bytes,omitempty"`
	RetainedBytes int64  `json:"retained_bytes,omitempty"`
}

type waldoAcquisition struct {
	Basis     string               `json:"basis,omitempty"`
	Crawler   *waldoCrawler        `json:"crawler,omitempty"`
	UserData  *waldoUserData       `json:"user_data,omitempty"`
	Synthetic *waldoSyntheticData  `json:"synthetic,omitempty"`
	Domains   []waldoDomainMeasure `json:"domains,omitempty"`
}

type waldoSource struct {
	Name            string                      `json:"name"`
	Source          string                      `json:"source"`
	Version         string                      `json:"version,omitempty"`
	License         string                      `json:"license,omitempty"`
	LicenseEvidence *waldoLicenseAssertion      `json:"license_evidence,omitempty"`
	URL             string                      `json:"url"`
	Category        string                      `json:"category,omitempty"`
	CollectedFrom   string                      `json:"collected_from,omitempty"`
	CollectedTo     string                      `json:"collected_to,omitempty"`
	SHA256          string                      `json:"sha256"`
	Files           []waldoSourceFile           `json:"files,omitempty"`
	Usage           map[string]ModalityEvidence `json:"usage,omitempty"`
	Content         *waldoContent               `json:"content,omitempty"`
	Acquisition     *waldoAcquisition           `json:"acquisition,omitempty"`
}

type waldoProcessingStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type waldoProcessing struct {
	Steps                     []waldoProcessingStep `json:"steps,omitempty"`
	RightsReservationMeasures []string              `json:"rights_reservation_measures,omitempty"`
	IllegalContentMeasures    []string              `json:"illegal_content_measures,omitempty"`
}

type waldoRecipeStep struct {
	Name       string `json:"name"`
	Executable string `json:"script"`
	SHA256     string `json:"sha256"`
}

type waldoRecipeEvidence struct {
	Path       string            `json:"path"`
	SHA256     string            `json:"sha256"`
	Repository string            `json:"repository,omitempty"`
	Commit     string            `json:"commit,omitempty"`
	Dirty      bool              `json:"dirty"`
	Steps      []waldoRecipeStep `json:"steps"`
}

type waldoCorpusManifest struct {
	Path         string                      `json:"path"`
	SHA256       string                      `json:"sha256"`
	Name         string                      `json:"name"`
	Title        string                      `json:"title"`
	Description  string                      `json:"description"`
	License      string                      `json:"license"`
	LicenseSet   []string                    `json:"license_set,omitempty"`
	Format       string                      `json:"format"`
	RecordSchema int                         `json:"record_schema"`
	ConvertedBy  waldoConversion             `json:"converted_by"`
	Sources      []waldoSource               `json:"sources"`
	Processing   *waldoProcessing            `json:"processing,omitempty"`
	ComposedBy   *waldoRecipeEvidence        `json:"composed_by,omitempty"`
	Totals       EvidenceMeasures            `json:"totals"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
	Licenses     map[string]EvidenceMeasures `json:"licenses"`
}

type waldoShardValidation struct {
	CanonicalRecords  bool `json:"canonical_records"`
	ContentHashes     bool `json:"content_hashes"`
	TokenCounts       bool `json:"token_counts"`
	ExactLicenseDedup bool `json:"exact_license_partition_dedup"`
}

type waldoShardBOM struct {
	Kind         string               `json:"kind"`
	Schema       int                  `json:"schema"`
	Subject      string               `json:"subject"`
	PlanSHA256   string               `json:"plan_sha256"`
	RecordSchema int                  `json:"record_schema"`
	WriterRecipe string               `json:"writer_recipe"`
	Tokenizer    string               `json:"tokenizer"`
	Records      int64                `json:"records"`
	Tokens       int64                `json:"tokens"`
	ContentBytes int64                `json:"content_bytes"`
	Licenses     []string             `json:"licenses"`
	Validation   waldoShardValidation `json:"validation"`
}

type waldoShardAttestation struct {
	Status       string         `json:"status"`
	WriterRecipe string         `json:"writer_recipe,omitempty"`
	BOMSHA256    string         `json:"bom_sha256,omitempty"`
	BOM          *waldoShardBOM `json:"bom,omitempty"`
}

type waldoCorpusShard struct {
	Manifest          string                      `json:"manifest"`
	SubManifestSHA256 string                      `json:"sub_manifest_sha256,omitempty"`
	URL               string                      `json:"url"`
	SHA256            string                      `json:"sha256"`
	Format            string                      `json:"format"`
	RecordSchema      int                         `json:"record_schema"`
	License           string                      `json:"license"`
	Licenses          []string                    `json:"licenses,omitempty"`
	LicenseUsage      map[string]EvidenceMeasures `json:"license_usage,omitempty"`
	Sources           []string                    `json:"sources,omitempty"`
	ConvertedBy       waldoConversion             `json:"converted_by"`
	RecordsRoot       string                      `json:"records_root,omitempty"`
	Docs              int64                       `json:"docs"`
	Tokens            int64                       `json:"tokens"`
	Bytes             int64                       `json:"bytes"`
	Modalities        map[string]ModalityEvidence `json:"modalities,omitempty"`
	Attestation       *waldoShardAttestation      `json:"attestation,omitempty"`
}

type corpusShardIdentity struct {
	Ordinal           int      `json:"ordinal"`
	Manifest          string   `json:"manifest"`
	SubManifestSHA256 string   `json:"sub_manifest_sha256,omitempty"`
	URL               string   `json:"url"`
	SHA256            string   `json:"sha256"`
	Format            string   `json:"format"`
	RecordSchema      int      `json:"record_schema"`
	Licenses          []string `json:"licenses"`
	Sources           []string `json:"sources,omitempty"`
	RecordsRoot       string   `json:"records_root,omitempty"`
	Docs              int64    `json:"docs"`
	Tokens            int64    `json:"tokens"`
	Bytes             int64    `json:"bytes"`
	AttestationStatus string   `json:"attestation_status,omitempty"`
	AttestationSHA256 string   `json:"attestation_sha256,omitempty"`
}

// LensCorpusBOM validates and projects a schema-1 corpus BOM. Unknown fields
// are accepted for WALDO's additive compatibility rule, while duplicate keys
// and trailing JSON values remain ambiguous and are rejected.
func LensCorpusBOM(data []byte) (CorpusEvidenceLens, error) {
	var bom waldoCorpusBOM
	if err := decodeJSON(data, &bom, false); err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("decode corpus BOM: %w", err)
	}
	if err := validateCorpusBOM(bom); err != nil {
		return CorpusEvidenceLens{}, err
	}

	canonical, err := json.Marshal(bom)
	if err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("canonicalize corpus BOM: %w", err)
	}
	manifests, sourceSetDigest, err := projectCorpusManifests(bom.Manifests)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	shardSetDigest, attestation, err := projectCorpusShards(bom.Shards)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}

	index := CorpusIndexEvidence{Remote: bom.Index.Remote, Commit: bom.Index.Commit, Dirty: bom.Index.Dirty}
	var notices []string
	switch {
	case bom.Index.Commit == "":
		index.State = "MANIFEST_PINNED_NO_COMMIT"
		notices = append(notices, "index revision is not recorded; manifest and shard digests remain the available identity pins")
	case bom.Index.Dirty:
		index.State = "DIRTY_MANIFEST_PINNED"
		notices = append(notices, "index checkout was dirty; the commit alone does not describe the selected manifest bytes")
	default:
		index.State = "CLEAN_PINNED"
	}
	notices = append(notices,
		"license values are recorded assertions and policy inputs, not legal conclusions",
		"hashes bind recorded identity and integrity; they do not prove truth, quality, safety, or actual trainer consumption",
	)

	lens := CorpusEvidenceLens{
		Schema:          CorpusEvidenceLensSchema,
		State:           CorpusEvidenceStateReady,
		BOMSHA256:       digestBytes(canonical),
		DocumentSHA256:  digestBytes(data),
		Index:           index,
		Paths:           append([]string(nil), bom.Paths...),
		LicensePolicy:   cloneLicensePolicy(bom.Policy),
		Manifests:       manifests,
		SourceSetSHA256: sourceSetDigest,
		ShardSetSHA256:  shardSetDigest,
		Totals:          bom.Totals,
		Modalities:      cloneModalitiesEvidence(bom.Modalities),
		Licenses:        projectCorpusLicenses(bom.Licenses),
		Attestation:     attestation,
		Notices:         notices,
		Authority:       Authority{},
	}
	lens.ReceiptSHA256, err = corpusLensReceiptDigest(lens)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	if err := lens.Validate(); err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("generated corpus evidence lens: %w", err)
	}
	return lens, nil
}

func (lens CorpusEvidenceLens) Validate() error {
	if lens.Schema != CorpusEvidenceLensSchema || lens.State != CorpusEvidenceStateReady {
		return fmt.Errorf("unsupported corpus evidence lens identity %q state %q", lens.Schema, lens.State)
	}
	for _, item := range []struct{ name, value string }{
		{"corpus lens.bom_sha256", lens.BOMSHA256},
		{"corpus lens.document_sha256", lens.DocumentSHA256},
		{"corpus lens.receipt_sha256", lens.ReceiptSHA256},
		{"corpus lens.source_set_sha256", lens.SourceSetSHA256},
		{"corpus lens.shard_set_sha256", lens.ShardSetSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if !oneOf(lens.Index.State, "CLEAN_PINNED", "DIRTY_MANIFEST_PINNED", "MANIFEST_PINNED_NO_COMMIT") {
		return fmt.Errorf("unsupported corpus index evidence state %q", lens.Index.State)
	}
	if lens.Index.Commit != "" && !validLowerGitHash(lens.Index.Commit) {
		return errors.New("corpus lens index commit is not a lowercase Git object hash")
	}
	if !sort.StringsAreSorted(lens.Paths) || hasAdjacentDuplicate(lens.Paths) {
		return errors.New("corpus lens paths must be sorted and unique")
	}
	for i, selected := range lens.Paths {
		if err := validateCorpusSelectionPath(fmt.Sprintf("corpus lens paths[%d]", i), selected); err != nil {
			return err
		}
	}
	if err := validateLicensePolicy(lens.LicensePolicy); err != nil {
		return err
	}
	expectedIndexState := "CLEAN_PINNED"
	if lens.Index.Commit == "" {
		expectedIndexState = "MANIFEST_PINNED_NO_COMMIT"
	} else if lens.Index.Dirty {
		expectedIndexState = "DIRTY_MANIFEST_PINNED"
	}
	if lens.Index.State != expectedIndexState {
		return fmt.Errorf("corpus index evidence state %q does not match its recorded identity", lens.Index.State)
	}
	if lens.Totals.Shards < 0 || lens.Totals.Docs < 0 || lens.Totals.Tokens < 0 || lens.Totals.Bytes < 0 {
		return errors.New("corpus lens has negative totals")
	}
	if err := validateModalities("corpus lens", lens.Modalities); err != nil {
		return err
	}
	previousManifest := ""
	for i, manifest := range lens.Manifests {
		if err := validateRelativePath(fmt.Sprintf("corpus lens manifests[%d].path", i), manifest.Path); err != nil {
			return err
		}
		if i > 0 && manifest.Path <= previousManifest {
			return errors.New("corpus lens manifests must be sorted and unique")
		}
		previousManifest = manifest.Path
		if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s.sha256", manifest.Path), manifest.SHA256); err != nil {
			return err
		}
		if strings.TrimSpace(manifest.Name) == "" || len(manifest.LicenseAssertions) == 0 {
			return fmt.Errorf("corpus lens manifest %s has incomplete identity or license assertions", manifest.Path)
		}
		if err := validateSortedNonemptySet("corpus lens manifest "+manifest.Path+" license assertions", manifest.LicenseAssertions); err != nil {
			return err
		}
		previousSource := ""
		for j, source := range manifest.Sources {
			if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Source) == "" || strings.TrimSpace(source.URL) == "" {
				return fmt.Errorf("corpus lens manifest %s source %d has incomplete identity", manifest.Path, j)
			}
			if j > 0 && source.Name <= previousSource {
				return fmt.Errorf("corpus lens manifest %s sources must be sorted and unique", manifest.Path)
			}
			previousSource = source.Name
			if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s source %s.sha256", manifest.Path, source.Name), source.SHA256); err != nil {
				return err
			}
			if source.SourceFileCount < 0 || source.SourceFileCount == 0 && source.SourceFilesSHA256 != "" || source.SourceFileCount > 0 && source.SourceFilesSHA256 == "" {
				return fmt.Errorf("corpus lens manifest %s source %s has inconsistent source-file evidence", manifest.Path, source.Name)
			}
			if source.SourceFilesSHA256 != "" {
				if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s source %s.source_files_sha256", manifest.Path, source.Name), source.SourceFilesSHA256); err != nil {
					return err
				}
			}
			switch source.License.State {
			case "MISSING":
				if source.License.Normalized != "" || source.License.Declaration != "" || source.License.EvidenceURL != "" {
					return fmt.Errorf("corpus lens manifest %s source %s marks present license evidence MISSING", manifest.Path, source.Name)
				}
			case "RECORDED_ASSERTION":
				if source.License.Normalized == "" && source.License.Declaration == "" && source.License.EvidenceURL == "" {
					return fmt.Errorf("corpus lens manifest %s source %s has an empty license assertion", manifest.Path, source.Name)
				}
			default:
				return fmt.Errorf("corpus lens manifest %s source %s has unsupported license state %q", manifest.Path, source.Name, source.License.State)
			}
		}
	}
	previousLicense := ""
	for i, license := range lens.Licenses {
		if strings.TrimSpace(license.ID) == "" || license.State != "RECORDED_ASSERTION" || license.Measures.Shards < 0 || license.Measures.Docs < 0 || license.Measures.Tokens < 0 || license.Measures.Bytes < 0 {
			return fmt.Errorf("corpus lens license %d is invalid", i)
		}
		if i > 0 && license.ID <= previousLicense {
			return errors.New("corpus lens licenses must be sorted and unique")
		}
		previousLicense = license.ID
	}
	if lens.Attestation.Embedded < 0 || lens.Attestation.ImplicitV4 < 0 || lens.Attestation.DeepValidated < 0 || lens.Attestation.NotRecorded < 0 {
		return errors.New("corpus lens has negative attestation counts")
	}
	attested := lens.Attestation.Embedded + lens.Attestation.ImplicitV4 + lens.Attestation.DeepValidated
	if attested+lens.Attestation.NotRecorded != lens.Totals.Shards {
		return errors.New("corpus lens attestation counts do not match its shard total")
	}
	expectedAttestationState := "RECORDED_COMPLETE"
	switch {
	case lens.Totals.Shards == 0:
		expectedAttestationState = "NO_SHARDS"
	case lens.Attestation.NotRecorded == lens.Totals.Shards:
		expectedAttestationState = "NOT_RECORDED"
	case lens.Attestation.NotRecorded > 0:
		expectedAttestationState = "RECORDED_PARTIAL"
	}
	if lens.Attestation.State != expectedAttestationState {
		return fmt.Errorf("corpus lens attestation state %q does not match its counts", lens.Attestation.State)
	}
	if len(lens.Notices) == 0 {
		return errors.New("corpus evidence lens must state its evidence boundaries")
	}
	if !lens.Authority.closed() {
		return errors.New("corpus evidence lens must carry closed authority")
	}
	actual, err := corpusLensReceiptDigest(lens)
	if err != nil {
		return err
	}
	if actual != lens.ReceiptSHA256 {
		return fmt.Errorf("corpus evidence lens receipt digest mismatch: expected %s, got %s", lens.ReceiptSHA256, actual)
	}
	return nil
}

func corpusLensReceiptDigest(lens CorpusEvidenceLens) (string, error) {
	lens.ReceiptSHA256 = ""
	data, err := json.Marshal(lens)
	if err != nil {
		return "", fmt.Errorf("encode corpus evidence lens receipt: %w", err)
	}
	return digestBytes(data), nil
}

func validateCorpusBOM(bom waldoCorpusBOM) error {
	if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "corpus" {
		return fmt.Errorf("unsupported corpus BOM identity %q schema %d subject %q", bom.Kind, bom.Schema, bom.Subject)
	}
	if bom.Index.Commit != "" && !validLowerGitHash(bom.Index.Commit) {
		return fmt.Errorf("corpus index commit %q is not a lowercase Git object hash", bom.Index.Commit)
	}
	if !sort.StringsAreSorted(bom.Paths) || hasAdjacentDuplicate(bom.Paths) {
		return errors.New("corpus BOM paths must be sorted and unique")
	}
	for i, selected := range bom.Paths {
		if err := validateCorpusSelectionPath(fmt.Sprintf("corpus paths[%d]", i), selected); err != nil {
			return err
		}
	}
	if err := validateLicensePolicy(bom.Policy); err != nil {
		return err
	}

	manifests := make(map[string]waldoCorpusManifest, len(bom.Manifests))
	manifestSources := make(map[string]map[string]bool, len(bom.Manifests))
	manifestTotals := make(map[string]EvidenceMeasures, len(bom.Manifests))
	manifestLicenses := make(map[string]map[string]EvidenceMeasures, len(bom.Manifests))
	manifestModalities := make(map[string]map[string]ModalityEvidence, len(bom.Manifests))
	for i, manifest := range bom.Manifests {
		if err := validateRelativePath(fmt.Sprintf("corpus manifests[%d].path", i), manifest.Path); err != nil {
			return err
		}
		if _, exists := manifests[manifest.Path]; exists {
			return fmt.Errorf("duplicate corpus manifest path %q", manifest.Path)
		}
		if err := validateSHA256(fmt.Sprintf("corpus manifest %s.sha256", manifest.Path), manifest.SHA256); err != nil {
			return err
		}
		if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Title) == "" || strings.TrimSpace(manifest.Description) == "" || strings.TrimSpace(manifest.Format) == "" || manifest.RecordSchema <= 0 {
			return fmt.Errorf("corpus manifest %s is missing resolved identity or format fields", manifest.Path)
		}
		if (manifest.License == "") == (len(manifest.LicenseSet) == 0) {
			return fmt.Errorf("corpus manifest %s must declare exactly one license or license_set form", manifest.Path)
		}
		if err := validateSortedNonemptySet("corpus manifest "+manifest.Path+" license_set", manifest.LicenseSet); err != nil {
			return err
		}
		if err := validateConversion("corpus manifest "+manifest.Path, manifest.ConvertedBy, manifest.RecordSchema); err != nil {
			return err
		}
		sources := map[string]bool{}
		for j, source := range manifest.Sources {
			if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Source) == "" || strings.TrimSpace(source.URL) == "" {
				return fmt.Errorf("corpus manifest %s source %d is missing name, source, or url", manifest.Path, j)
			}
			if sources[source.Name] {
				return fmt.Errorf("corpus manifest %s has duplicate source %q", manifest.Path, source.Name)
			}
			sources[source.Name] = true
			if err := validateSHA256(fmt.Sprintf("corpus manifest %s source %s.sha256", manifest.Path, source.Name), source.SHA256); err != nil {
				return err
			}
			for k, file := range source.Files {
				if strings.TrimSpace(file.Name) == "" || strings.TrimSpace(file.URL) == "" {
					return fmt.Errorf("corpus manifest %s source %s file %d is missing name or url", manifest.Path, source.Name, k)
				}
				if err := validateSHA256(fmt.Sprintf("corpus manifest %s source %s file %d.sha256", manifest.Path, source.Name, k), file.SHA256); err != nil {
					return err
				}
				if file.Bytes < 0 {
					return fmt.Errorf("corpus manifest %s source %s file %d has negative bytes", manifest.Path, source.Name, k)
				}
			}
			if err := validateModalities("corpus manifest "+manifest.Path+" source "+source.Name+" usage", source.Usage); err != nil {
				return err
			}
		}
		if err := validateModalities("corpus manifest "+manifest.Path, manifest.Modalities); err != nil {
			return err
		}
		if err := validateMeasureMap("corpus manifest "+manifest.Path+" licenses", manifest.Licenses); err != nil {
			return err
		}
		manifests[manifest.Path] = manifest
		manifestSources[manifest.Path] = sources
		manifestLicenses[manifest.Path] = map[string]EvidenceMeasures{}
		manifestModalities[manifest.Path] = map[string]ModalityEvidence{}
	}

	subManifestKeys, err := validateCorpusSubManifests(bom.SubManifests, manifests)
	if err != nil {
		return err
	}
	calculated := EvidenceMeasures{}
	calculatedLicenses := map[string]EvidenceMeasures{}
	calculatedModalities := map[string]ModalityEvidence{}
	for i, shard := range bom.Shards {
		manifest, exists := manifests[shard.Manifest]
		if !exists {
			return fmt.Errorf("corpus shard %d refers to unknown manifest %q", i+1, shard.Manifest)
		}
		if strings.TrimSpace(shard.URL) == "" || strings.TrimSpace(shard.Format) == "" || shard.RecordSchema <= 0 {
			return fmt.Errorf("corpus shard %d has incomplete resolved identity", i+1)
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d.sha256", i+1), shard.SHA256); err != nil {
			return err
		}
		if (shard.License == "") == (len(shard.Licenses) == 0) {
			return fmt.Errorf("corpus shard %d must declare exactly one license or licenses form", i+1)
		}
		if err := validateSortedNonemptySet(fmt.Sprintf("corpus shard %d licenses", i+1), shard.Licenses); err != nil {
			return err
		}
		if err := validateConversion(fmt.Sprintf("corpus shard %d", i+1), shard.ConvertedBy, shard.RecordSchema); err != nil {
			return err
		}
		if shard.RecordsRoot != "" {
			if err := validateSHA256(fmt.Sprintf("corpus shard %d.records_root", i+1), shard.RecordsRoot); err != nil {
				return err
			}
		}
		if shard.SubManifestSHA256 != "" && !subManifestKeys[shard.Manifest+"\x00"+shard.SubManifestSHA256] {
			return fmt.Errorf("corpus shard %d refers to an unpinned submanifest", i+1)
		}
		if shard.Docs <= 0 || shard.Tokens < 0 || shard.Bytes <= 0 {
			return fmt.Errorf("corpus shard %d has non-positive totals", i+1)
		}
		if err := validateModalities(fmt.Sprintf("corpus shard %d", i+1), shard.Modalities); err != nil {
			return err
		}
		if len(shard.Modalities) > 0 && modalityTokens(shard.Modalities) != shard.Tokens {
			return fmt.Errorf("corpus shard %d modality tokens do not match its token total", i+1)
		}
		seenSources := map[string]bool{}
		for _, source := range shard.Sources {
			if !manifestSources[shard.Manifest][source] || seenSources[source] {
				return fmt.Errorf("corpus shard %d has unknown or duplicate source %q", i+1, source)
			}
			seenSources[source] = true
		}
		licenses := effectiveShardLicenses(shard)
		for _, license := range licenses {
			if !licensePolicyAllows(bom.Policy, license) {
				return fmt.Errorf("corpus shard %d license %q violates the BOM policy", i+1, license)
			}
		}
		if err := validateLicenseUsage(fmt.Sprintf("corpus shard %d", i+1), shard, licenses); err != nil {
			return err
		}
		if err := validateCorpusAttestation(i+1, shard, licenses); err != nil {
			return err
		}

		measure := EvidenceMeasures{Shards: 1, Docs: shard.Docs, Tokens: shard.Tokens, Bytes: shard.Bytes}
		addEvidenceMeasures(&calculated, measure)
		addCorpusLicenseMeasures(calculatedLicenses, shard, licenses, measure)
		addModalityMeasures(calculatedModalities, shard.Modalities)
		manifestMeasure := manifestTotals[manifest.Path]
		addEvidenceMeasures(&manifestMeasure, measure)
		manifestTotals[manifest.Path] = manifestMeasure
		addCorpusLicenseMeasures(manifestLicenses[manifest.Path], shard, licenses, measure)
		addModalityMeasures(manifestModalities[manifest.Path], shard.Modalities)
	}
	if calculated != bom.Totals || !maps.Equal(calculatedLicenses, bom.Licenses) || !maps.Equal(calculatedModalities, bom.Modalities) {
		return errors.New("corpus BOM totals, modality totals, or license totals do not match its shards")
	}
	for path, manifest := range manifests {
		if manifest.Totals != manifestTotals[path] || !maps.Equal(manifest.Licenses, manifestLicenses[path]) || !maps.Equal(manifest.Modalities, manifestModalities[path]) {
			return fmt.Errorf("corpus manifest %s totals do not match its selected shards", path)
		}
	}
	return nil
}

func validateCorpusSubManifests(pins []waldoCorpusSubManifest, manifests map[string]waldoCorpusManifest) (map[string]bool, error) {
	seen := map[string]bool{}
	parents := map[string]string{}
	roots := map[string]int{}
	for i, pin := range pins {
		if _, exists := manifests[pin.Manifest]; !exists {
			return nil, fmt.Errorf("corpus submanifest %d refers to unknown manifest %q", i+1, pin.Manifest)
		}
		key := pin.Manifest + "\x00" + pin.SHA256
		if seen[key] || strings.TrimSpace(pin.URL) == "" {
			return nil, fmt.Errorf("invalid or duplicate corpus submanifest %d", i+1)
		}
		if err := validateSHA256(fmt.Sprintf("corpus submanifest %d.sha256", i+1), pin.SHA256); err != nil {
			return nil, err
		}
		if pin.ParentSHA256 != "" {
			if err := validateSHA256(fmt.Sprintf("corpus submanifest %d.parent_sha256", i+1), pin.ParentSHA256); err != nil {
				return nil, err
			}
		}
		if pin.Count <= 0 || pin.Docs <= 0 || pin.Tokens < 0 || pin.Bytes <= 0 || pin.EncodedBytes <= 0 {
			return nil, fmt.Errorf("corpus submanifest %d has non-positive totals", i+1)
		}
		if err := validateModalities(fmt.Sprintf("corpus submanifest %d", i+1), pin.Modalities); err != nil {
			return nil, err
		}
		if len(pin.Modalities) > 0 && modalityTokens(pin.Modalities) != pin.Tokens {
			return nil, fmt.Errorf("corpus submanifest %d modality tokens do not match its token total", i+1)
		}
		seen[key] = true
		if pin.ParentSHA256 == "" {
			roots[pin.Manifest]++
		} else {
			parents[key] = pin.Manifest + "\x00" + pin.ParentSHA256
		}
	}
	for key, parent := range parents {
		if !seen[parent] {
			return nil, fmt.Errorf("corpus submanifest %s refers to an unpinned parent", strings.Split(key, "\x00")[1])
		}
		chain := map[string]bool{}
		for current := key; current != ""; current = parents[current] {
			if chain[current] {
				return nil, errors.New("corpus submanifest parent cycle")
			}
			chain[current] = true
		}
	}
	for manifest, count := range roots {
		if count != 1 {
			return nil, fmt.Errorf("corpus manifest %s has %d submanifest roots, want one", manifest, count)
		}
	}
	return seen, nil
}

func validateCorpusAttestation(position int, shard waldoCorpusShard, licenses []string) error {
	attestation := shard.Attestation
	if attestation == nil {
		return nil
	}
	switch attestation.Status {
	case "embedded":
		if attestation.BOM == nil {
			return fmt.Errorf("corpus shard %d has incomplete embedded BOM evidence", position)
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d attestation.bom_sha256", position), attestation.BOMSHA256); err != nil {
			return err
		}
		bom := attestation.BOM
		if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "shard" || bom.WriterRecipe != waldoTextWriterRecipe || strings.TrimSpace(bom.Tokenizer) == "" || bom.RecordSchema != waldoTextRecordSchema || bom.RecordSchema != shard.RecordSchema || bom.Records != shard.Docs || bom.Tokens != shard.Tokens || !slices.Equal(bom.Licenses, licenses) {
			return fmt.Errorf("corpus shard %d embedded BOM differs from its corpus pin", position)
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d embedded plan_sha256", position), bom.PlanSHA256); err != nil {
			return err
		}
		if bom.ContentBytes <= 0 || !bom.Validation.CanonicalRecords || !bom.Validation.ContentHashes || !bom.Validation.TokenCounts || !bom.Validation.ExactLicenseDedup || attestation.WriterRecipe != bom.WriterRecipe {
			return fmt.Errorf("corpus shard %d embedded BOM has incomplete validation evidence", position)
		}
		encoded, err := json.Marshal(bom)
		if err != nil {
			return err
		}
		if digestBytes(encoded) != attestation.BOMSHA256 {
			return fmt.Errorf("corpus shard %d embedded BOM digest differs", position)
		}
	case "implicit-v4":
		if attestation.WriterRecipe != waldoFormerTextRecipe || attestation.BOM != nil || attestation.BOMSHA256 != "" {
			return fmt.Errorf("corpus shard %d has invalid implicit-v4 evidence", position)
		}
	case "deep-validated":
		if attestation.BOM != nil || attestation.BOMSHA256 != "" {
			return fmt.Errorf("corpus shard %d has invalid deep-validation evidence", position)
		}
	default:
		return fmt.Errorf("corpus shard %d has unsupported attestation status %q", position, attestation.Status)
	}
	return nil
}

func validateLicenseUsage(name string, shard waldoCorpusShard, licenses []string) error {
	if len(licenses) > 1 {
		if len(shard.LicenseUsage) != len(licenses) {
			return fmt.Errorf("%s license usage must cover every represented license", name)
		}
		var docs, tokens int64
		for _, license := range licenses {
			usage, exists := shard.LicenseUsage[license]
			if !exists || usage.Docs <= 0 || usage.Tokens < 0 || usage.Shards != 0 || usage.Bytes != 0 {
				return fmt.Errorf("%s has invalid usage for license %q", name, license)
			}
			docs += usage.Docs
			tokens += usage.Tokens
		}
		if docs != shard.Docs || tokens != shard.Tokens {
			return fmt.Errorf("%s license usage does not match shard totals", name)
		}
		return nil
	}
	if len(shard.LicenseUsage) == 0 {
		return nil
	}
	usage, exists := shard.LicenseUsage[licenses[0]]
	if len(shard.LicenseUsage) != 1 || !exists || usage.Docs != shard.Docs || usage.Tokens != shard.Tokens || usage.Shards != 0 || usage.Bytes != 0 {
		return fmt.Errorf("%s has invalid usage for license %q", name, licenses[0])
	}
	return nil
}

func projectCorpusManifests(input []waldoCorpusManifest) ([]CorpusManifestEvidence, string, error) {
	output := make([]CorpusManifestEvidence, 0, len(input))
	for _, manifest := range input {
		licenses := append([]string(nil), manifest.LicenseSet...)
		if manifest.License != "" {
			licenses = []string{manifest.License}
		}
		sources := make([]CorpusSourceEvidence, 0, len(manifest.Sources))
		for _, source := range manifest.Sources {
			license := SourceLicenseAssertion{Normalized: source.License, State: "MISSING"}
			if source.LicenseEvidence != nil {
				license.Declaration = source.LicenseEvidence.Declaration
				license.EvidenceURL = source.LicenseEvidence.URL
			}
			if license.Normalized != "" || license.Declaration != "" || license.EvidenceURL != "" {
				license.State = "RECORDED_ASSERTION"
			}
			sourceEvidence := CorpusSourceEvidence{
				Name: source.Name, Source: source.Source, Version: source.Version, URL: source.URL,
				SHA256: source.SHA256, License: license, SourceFileCount: len(source.Files),
			}
			if len(source.Files) > 0 {
				encoded, err := json.Marshal(source.Files)
				if err != nil {
					return nil, "", fmt.Errorf("encode source files for %s/%s: %w", manifest.Path, source.Name, err)
				}
				sourceEvidence.SourceFilesSHA256 = digestBytes(encoded)
			}
			sources = append(sources, sourceEvidence)
		}
		sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
		output = append(output, CorpusManifestEvidence{Path: manifest.Path, SHA256: manifest.SHA256, Name: manifest.Name, LicenseAssertions: licenses, Sources: sources})
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Path < output[j].Path })
	encoded, err := json.Marshal(output)
	if err != nil {
		return nil, "", fmt.Errorf("encode corpus source evidence: %w", err)
	}
	return output, digestBytes(encoded), nil
}

func projectCorpusShards(shards []waldoCorpusShard) (string, CorpusAttestationEvidence, error) {
	identities := make([]corpusShardIdentity, 0, len(shards))
	attestation := CorpusAttestationEvidence{}
	for i, shard := range shards {
		identity := corpusShardIdentity{
			Ordinal: i + 1, Manifest: shard.Manifest, SubManifestSHA256: shard.SubManifestSHA256,
			URL: shard.URL, SHA256: shard.SHA256, Format: shard.Format, RecordSchema: shard.RecordSchema,
			Licenses: append([]string(nil), effectiveShardLicenses(shard)...), Sources: append([]string(nil), shard.Sources...),
			RecordsRoot: shard.RecordsRoot, Docs: shard.Docs, Tokens: shard.Tokens, Bytes: shard.Bytes,
		}
		if shard.Attestation == nil {
			attestation.NotRecorded++
		} else {
			identity.AttestationStatus = shard.Attestation.Status
			identity.AttestationSHA256 = shard.Attestation.BOMSHA256
			switch shard.Attestation.Status {
			case "embedded":
				attestation.Embedded++
			case "implicit-v4":
				attestation.ImplicitV4++
			case "deep-validated":
				attestation.DeepValidated++
			}
		}
		identities = append(identities, identity)
	}
	switch {
	case len(shards) == 0:
		attestation.State = "NO_SHARDS"
	case attestation.NotRecorded == int64(len(shards)):
		attestation.State = "NOT_RECORDED"
	case attestation.NotRecorded > 0:
		attestation.State = "RECORDED_PARTIAL"
	default:
		attestation.State = "RECORDED_COMPLETE"
	}
	encoded, err := json.Marshal(identities)
	if err != nil {
		return "", CorpusAttestationEvidence{}, fmt.Errorf("encode corpus shard identities: %w", err)
	}
	return digestBytes(encoded), attestation, nil
}

func projectCorpusLicenses(input map[string]EvidenceMeasures) []CorpusLicenseEvidence {
	licenses := make([]CorpusLicenseEvidence, 0, len(input))
	for id, measure := range input {
		licenses = append(licenses, CorpusLicenseEvidence{ID: id, State: "RECORDED_ASSERTION", Measures: measure})
	}
	sort.Slice(licenses, func(i, j int) bool { return licenses[i].ID < licenses[j].ID })
	return licenses
}

func validateLicensePolicy(policy CorpusLicensePolicy) error {
	for label, values := range map[string][]string{"include": policy.Include, "exclude": policy.Exclude} {
		seen := map[string]bool{}
		for _, pattern := range values {
			if pattern == "" || seen[pattern] {
				return fmt.Errorf("corpus license policy %s patterns must be non-empty and unique", label)
			}
			seen[pattern] = true
			if _, err := pathpkg.Match(pattern, "probe"); err != nil {
				return fmt.Errorf("invalid corpus license pattern %q: %w", pattern, err)
			}
		}
	}
	return nil
}

func licensePolicyAllows(policy CorpusLicensePolicy, license string) bool {
	for _, pattern := range policy.Exclude {
		if matched, _ := pathpkg.Match(pattern, license); matched {
			return false
		}
	}
	if len(policy.Include) == 0 {
		return true
	}
	for _, pattern := range policy.Include {
		if matched, _ := pathpkg.Match(pattern, license); matched {
			return true
		}
	}
	return false
}

func validateConversion(name string, conversion waldoConversion, recordSchema int) error {
	if strings.TrimSpace(conversion.Tool) == "" || strings.TrimSpace(conversion.Version) == "" || strings.TrimSpace(conversion.Profile) == "" || strings.TrimSpace(conversion.Recipe) == "" || recordSchema != 1 && strings.TrimSpace(conversion.Tokenizer) == "" {
		return fmt.Errorf("%s has incomplete conversion provenance", name)
	}
	return nil
}

func validateSortedNonemptySet(name string, values []string) error {
	if len(values) == 0 {
		return nil
	}
	if !sort.StringsAreSorted(values) || hasAdjacentDuplicate(values) {
		return fmt.Errorf("%s must be sorted and unique", name)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s contains an empty value", name)
		}
	}
	return nil
}

func validateMeasureMap(name string, values map[string]EvidenceMeasures) error {
	for id, measure := range values {
		if strings.TrimSpace(id) == "" || measure.Shards < 0 || measure.Docs < 0 || measure.Tokens < 0 || measure.Bytes < 0 {
			return fmt.Errorf("%s contains an invalid measure for %q", name, id)
		}
	}
	return nil
}

func validateModalities(name string, modalities map[string]ModalityEvidence) error {
	for modality, measure := range modalities {
		if strings.TrimSpace(modality) == "" || measure.Samples < 0 || measure.Items < 0 || measure.Tokens < 0 || measure.DurationMS < 0 || measure.ContentBytes < 0 || measure == (ModalityEvidence{}) {
			return fmt.Errorf("%s contains invalid modality %q", name, modality)
		}
	}
	return nil
}

func validateCorpusSelectionPath(name, value string) error {
	if value == "" { // WALDO uses the empty string for an index-root selection.
		return nil
	}
	return validateRelativePath(name, value)
}

func validLowerGitHash(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func hasAdjacentDuplicate(values []string) bool {
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			return true
		}
	}
	return false
}

func effectiveShardLicenses(shard waldoCorpusShard) []string {
	if len(shard.Licenses) > 0 {
		return shard.Licenses
	}
	return []string{shard.License}
}

func addEvidenceMeasures(target *EvidenceMeasures, value EvidenceMeasures) {
	target.Shards += value.Shards
	target.Docs += value.Docs
	target.Tokens += value.Tokens
	target.Bytes += value.Bytes
}

func addCorpusLicenseMeasures(target map[string]EvidenceMeasures, shard waldoCorpusShard, licenses []string, fallback EvidenceMeasures) {
	if len(licenses) == 1 || len(shard.LicenseUsage) == 0 {
		value := target[licenses[0]]
		addEvidenceMeasures(&value, fallback)
		target[licenses[0]] = value
		return
	}
	for _, license := range licenses {
		usage := shard.LicenseUsage[license]
		usage.Shards = 1
		value := target[license]
		addEvidenceMeasures(&value, usage)
		target[license] = value
	}
}

func addModalityMeasures(target, values map[string]ModalityEvidence) {
	for name, value := range values {
		current := target[name]
		current.Samples += value.Samples
		current.Items += value.Items
		current.Tokens += value.Tokens
		current.DurationMS += value.DurationMS
		current.ContentBytes += value.ContentBytes
		target[name] = current
	}
}

func modalityTokens(modalities map[string]ModalityEvidence) int64 {
	var tokens int64
	for _, value := range modalities {
		tokens += value.Tokens
	}
	return tokens
}

func cloneLicensePolicy(input CorpusLicensePolicy) CorpusLicensePolicy {
	return CorpusLicensePolicy{Include: append([]string(nil), input.Include...), Exclude: append([]string(nil), input.Exclude...)}
}

func cloneModalitiesEvidence(input map[string]ModalityEvidence) map[string]ModalityEvidence {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]ModalityEvidence, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
