package pipeline

import (
	"crypto/x509"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// SelectCertPool creates a new x509.CertPool from all certificates in the loaded TSLs.
// This step processes all TSLs in the context's TSL stack and extracts certificates
// from trust service providers, adding them to a new certificate pool.
//
// The function walks through each TSL's trust service providers and their services,
// collecting all valid X.509 certificates. These certificates are then added to a new
// certificate pool that can be used for certificate chain validation.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: Optional arguments:
//   - "reference-depth:N": Process TSLs up to N levels deep in references (0=root only, 1=root+direct refs)
//   - "include-referenced": Legacy option, equivalent to a large reference depth (includes all refs)
//   - "service-type:URI": Filter certificates by service type URI (can be provided multiple times)
//   - "status:URI": Filter certificates by status URI (can be provided multiple times)
//   - "status-logic:and": Use AND logic for status filters (all filters must match) instead of default OR logic
//
// Returns:
//   - *Context: Updated context with the new certificate pool in ctx.CertPool
//   - error: Non-nil if no TSLs are loaded or if certificate processing fails
//
// The created certificate pool is stored in the context's CertPool field and can be
// used for certificate validation operations. Each certificate from valid trust services
// is added as a trusted root certificate.
//
// Note:
//   - Requires at least one TSL to be loaded in the context
//   - Invalid or nil TSLs in the stack are safely skipped
//   - The previous certificate pool, if any, is replaced
//   - The reference-depth parameter controls how deep in the TSL reference tree to process
//   - Service type and status filters are combined with OR logic within each category and AND between categories
//
// Example usage in pipeline configuration:
//   - select  # Create cert pool from top TSL only, all service types
//   - select: [reference-depth:1]  # Include root and direct references only
//   - select: [reference-depth:2]  # Include root, direct refs, and refs of refs (2 levels)
//   - select: [include-referenced]  # Legacy option: include all references
//   - select: ["service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC"]  # Only qualified CA certificates
//   - select: ["reference-depth:1", "service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC", "status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"]  # Only granted qualified CA certificates up to depth 1
//   - select: ["status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/", "status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/recognized/", "status-logic:and"]  # Only certificates that match both status filters
func SelectCertPool(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	// Check if we have TSLs either in the legacy stack or in the tree structure
	if (ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty()) && (ctx.TSLs == nil || ctx.TSLs.IsEmpty()) {
		return ctx, fmt.Errorf("no TSLs loaded")
	}

	// Parse arguments
	referenceDepth := 0 // Default: only root TSLs (no references)
	serviceTypeFilters := []string{}
	statusFilters := []string{}
	useStatusAndLogic := false // Default: use OR logic for status filters

	for _, arg := range args {
		if arg == "include-referenced" {
			// Legacy option: set depth to a large number to include all references
			referenceDepth = 100
		} else if strings.HasPrefix(arg, "reference-depth:") {
			depthStr := strings.TrimPrefix(arg, "reference-depth:")
			if depth, err := strconv.Atoi(depthStr); err == nil && depth >= 0 {
				referenceDepth = depth
			} else if err != nil {
				pl.Logger.Warn("Invalid reference-depth value, using default",
					logging.F("value", depthStr),
					logging.F("default", referenceDepth))
			}
		} else if strings.HasPrefix(arg, "service-type:") {
			serviceType := strings.TrimPrefix(arg, "service-type:")
			if serviceType != "" {
				serviceTypeFilters = append(serviceTypeFilters, serviceType)
			}
		} else if strings.HasPrefix(arg, "status:") {
			status := strings.TrimPrefix(arg, "status:")
			if status != "" {
				statusFilters = append(statusFilters, status)
			}
		} else if arg == "status-logic:and" {
			useStatusAndLogic = true
		}
	}

	// Initialize the certificate pool
	ctx.InitCertPool()

	// Track certificate counts for logging
	certCount := 0
	tslCount := 0

	// Create a certificate processing function that applies filters
	processCertificate := func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType, cert *x509.Certificate) {
		// Apply service type filter if specified
		if len(serviceTypeFilters) > 0 {
			serviceTypeMatch := false
			serviceType := svc.TslServiceInformation.TslServiceTypeIdentifier
			for _, filter := range serviceTypeFilters {
				if serviceType == filter {
					serviceTypeMatch = true
					break
				}
			}
			if !serviceTypeMatch {
				return
			}
		}

		// Apply status filter if specified
		if len(statusFilters) > 0 {
			status := svc.TslServiceInformation.TslServiceStatus

			if useStatusAndLogic {
				// AND logic: certificate must match ALL status filters
				for _, filter := range statusFilters {
					if status != filter {
						// If any filter doesn't match, skip this certificate
						return
					}
				}
			} else {
				// OR logic (default): certificate must match ANY status filter
				statusMatch := false
				for _, filter := range statusFilters {
					if status == filter {
						statusMatch = true
						break
					}
				}
				if !statusMatch {
					return
				}
			}
		}

		// Add the certificate to the pool
		ctx.CertPool.AddCert(cert)
		certCount++
	}

	// Define a function to process a TSL and extract certificates
	processTSL := func(tsl *etsi119612.TSL) {
		if tsl == nil {
			return
		}

		tslCount++

		// Process the TSL
		tsl.WithTrustServices(func(tsp *etsi119612.TSPType, svc *etsi119612.TSPServiceType) {
			svc.WithCertificates(func(cert *x509.Certificate) {
				processCertificate(tsp, svc, cert)
			})
		})
	}

	// Define a function to process a tree with a limited depth
	processTreeWithDepth := func(tree *TSLTree, processFunc func(*etsi119612.TSL), maxDepth int) {
		if tree == nil || tree.Root == nil || maxDepth < 0 {
			return
		}

		// Process nodes recursively with depth tracking
		var processNodeWithDepth func(node *TSLNode, currentDepth int)
		processNodeWithDepth = func(node *TSLNode, currentDepth int) {
			if node == nil || currentDepth > maxDepth {
				return
			}

			// Process this node's TSL
			processFunc(node.TSL)

			// Process children up to maxDepth
			for _, childNode := range node.Children {
				processNodeWithDepth(childNode, currentDepth+1)
			}
		}

		// Start processing from the root at depth 0
		processNodeWithDepth(tree.Root, 0)
	}

	// Check if we should use the legacy stack
	if ctx.TSLs != nil && !ctx.TSLs.IsEmpty() {
		// Process TSLs from the legacy stack
		tsls := ctx.TSLs.ToSlice()
		for i, tsl := range tsls {
			if tsl == nil {
				continue
			}

			// In legacy mode, with a flat list:
			// - The root TSL is at index 0
			// - Referenced TSLs come after, but we don't have depth information
			// - So we'll include TSLs up to the reference depth
			if i == 0 || (i > 0 && i <= referenceDepth) {
				processTSL(tsl)
			}
		}
	} else {
		// Process each TSL tree in the stack
		treeSlice := ctx.TSLTrees.ToSlice()
		for _, tree := range treeSlice {
			if tree == nil || tree.Root == nil {
				continue
			}

			if referenceDepth > 0 {
				// Process TSLs up to the specified reference depth
				processTreeWithDepth(tree, processTSL, referenceDepth)
			} else {
				// Process only the root TSL
				processTSL(tree.Root.TSL)
			}
		}
	}

	// Log summary information
	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Certificate pool created",
			logging.F("tsl_count", tslCount),
			logging.F("certificate_count", certCount),
			logging.F("reference_depth", referenceDepth),
			logging.F("service_type_filters", len(serviceTypeFilters)),
			logging.F("status_filters", len(statusFilters)))
	}

	if pl != nil && pl.Logger != nil {
		if len(serviceTypeFilters) > 0 {
			pl.Logger.Debug("Service type filters applied",
				logging.F("filters", serviceTypeFilters))
		}

		if len(statusFilters) > 0 {
			pl.Logger.Debug("Status filters applied",
				logging.F("filters", statusFilters))
		}
	}

	return ctx, nil
}
