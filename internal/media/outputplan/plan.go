package outputplan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/maucabreradev/video-compressor-extended/internal/media/pathutil"
)

// OutputPlan describes the final video and subtitle destinations for one input.
type OutputPlan struct {
	Source   string
	Video    string
	Subtitle string
}

// PlanOutputs maps each input to an MP4 and SRT destination while preserving
// directory structure. It returns an error if outputs collide with each other
// or any selected input.
func PlanOutputs(inputRoot, outputRoot string, inputs []string) ([]OutputPlan, error) {
	if strings.TrimSpace(inputRoot) == "" {
		return nil, fmt.Errorf("input root must not be empty")
	}
	if strings.TrimSpace(outputRoot) == "" {
		return nil, fmt.Errorf("output root must not be empty")
	}

	inputRootAbs, err := absolutePath(inputRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve input root %q: %w", inputRoot, err)
	}
	resolvedInputRoot, err := pathutil.Resolve(inputRootAbs)
	if err != nil {
		return nil, fmt.Errorf("resolve input root %q: %w", inputRoot, err)
	}
	outputRootAbs, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve output root %q: %w", outputRoot, err)
	}
	outputRootAbs = filepath.Clean(outputRootAbs)
	resolvedOutputRoot, err := pathutil.Resolve(outputRootAbs)
	if err != nil {
		return nil, fmt.Errorf("resolve output root %q: %w", outputRoot, err)
	}
	rootInfo, err := os.Stat(inputRootAbs)
	if err != nil {
		return nil, fmt.Errorf("inspect input root %q: %w", inputRootAbs, err)
	}

	uniqueInputs := make(map[string]string, len(inputs))
	for _, input := range inputs {
		path, err := absolutePath(input)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", input, err)
		}
		resolvedPath, err := pathutil.Resolve(path)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", input, err)
		}
		uniqueInputs[pathKey(resolvedPath)] = path
	}
	orderedInputs := make([]string, 0, len(uniqueInputs))
	for _, input := range uniqueInputs {
		orderedInputs = append(orderedInputs, input)
	}
	sort.Strings(orderedInputs)

	inputKeys := make(map[string]string, len(orderedInputs))
	for _, input := range orderedInputs {
		resolvedInput, err := pathutil.Resolve(input)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", input, err)
		}
		info, err := os.Stat(input)
		if err != nil {
			return nil, fmt.Errorf("inspect input %q: %w", input, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("input %q is not a regular file", input)
		}
		if isWithin(resolvedOutputRoot, resolvedInput) {
			return nil, fmt.Errorf("input %q is inside output root %q", input, resolvedOutputRoot)
		}
		if rootInfo.IsDir() {
			if !isWithin(resolvedInputRoot, resolvedInput) || pathsEqual(resolvedInputRoot, resolvedInput) {
				return nil, fmt.Errorf("input %q is outside input root %q", input, inputRootAbs)
			}
		} else if !pathsEqual(resolvedInputRoot, resolvedInput) {
			return nil, fmt.Errorf("input %q does not match file input root %q", input, inputRootAbs)
		}
		inputKeys[pathKey(resolvedInput)] = input
	}

	plans := make([]OutputPlan, 0, len(orderedInputs))
	outputOwners := make(map[string]string, len(orderedInputs)*2)
	for _, input := range orderedInputs {
		relative, err := relativeInputPath(inputRootAbs, rootInfo.IsDir(), input)
		if err != nil {
			return nil, err
		}
		base := strings.TrimSuffix(relative, filepath.Ext(relative))
		video := filepath.Join(outputRootAbs, base+".mp4")
		subtitle := filepath.Join(outputRootAbs, base+".srt")

		for _, destination := range []string{video, subtitle} {
			resolvedDestination, err := pathutil.Resolve(destination)
			if err != nil {
				return nil, fmt.Errorf("resolve output %q: %w", destination, err)
			}
			key := pathKey(resolvedDestination)
			if source, ok := inputKeys[key]; ok {
				return nil, fmt.Errorf("output %q would overwrite input %q", destination, source)
			}
			if previousSource, ok := outputOwners[key]; ok {
				return nil, fmt.Errorf("output collision at %q between inputs %q and %q", destination, previousSource, input)
			}
			outputOwners[key] = input
		}
		plans = append(plans, OutputPlan{Source: input, Video: video, Subtitle: subtitle})
	}
	return plans, nil
}

func absolutePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func relativeInputPath(inputRoot string, rootIsDirectory bool, input string) (string, error) {
	if !rootIsDirectory {
		return filepath.Base(input), nil
	}
	relative, err := filepath.Rel(inputRoot, input)
	if err != nil {
		return "", fmt.Errorf("make input %q relative to %q: %w", input, inputRoot, err)
	}
	if relative == "." || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("input %q is outside input root %q", input, inputRoot)
	}
	return relative, nil
}

func pathsEqual(first, second string) bool {
	first = filepath.Clean(first)
	second = filepath.Clean(second)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(first, second)
	}
	return first == second
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}
