<?xml version="1.0" encoding="UTF-8"?>
<!--
  TSL to HTML Stylesheet
  
  This XSLT stylesheet transforms an ETSI TS 119 612 Trust Status List (TSL) 
  into a comprehensive HTML representation for easy viewing and analysis.
  
  Compatible with ETSI TS 119 612 v2.1.1 and v2.2.1
-->
<xsl:stylesheet version="1.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:tsl="http://uri.etsi.org/02231/v2#"
  xmlns:ns2="http://www.w3.org/2000/09/xmldsig#"
  xmlns:ns3="http://uri.etsi.org/02231/v2/additionaltypes#"
  xmlns:ns4="http://uri.etsi.org/01903/v1.3.2#"
  xmlns:ns5="http://uri.etsi.org/TrstSvc/SvcInfoExt/eSigDir-1999-93-EC-TrustedList/#"
  exclude-result-prefixes="tsl ns2 ns3 ns4 ns5">

  <xsl:output method="html" encoding="UTF-8" indent="yes" doctype-system="about:legacy-compat"/>
  
  <!-- Main template -->
  <xsl:template match="/">
    <html lang="en">
      <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>
          <xsl:value-of select="tsl:TrustServiceStatusList/tsl:SchemeInformation/tsl:SchemeTerritory"/>
          <xsl:text> - Trust Service Status List</xsl:text>
        </title>
        <style>
          /* Design Tokens - Matching g119612 design system */
          :root {
            --color-primary: #0066cc;
            --color-primary-dark: #004d99;
            --color-success: #28a745;
            --color-warning: #f57c00;
            --color-error: #d32f2f;
            --color-bg: #f8f9fa;
            --color-surface: #ffffff;
            --color-text: #212529;
            --color-text-muted: #6c757d;
            --color-border: #dee2e6;
            --font-sans: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            --font-mono: 'SF Mono', Monaco, 'Cascadia Code', 'Courier New', monospace;
            --radius: 8px;
            --radius-sm: 4px;
            --shadow: 0 2px 4px rgba(0,0,0,0.1);
          }
          *, *::before, *::after { box-sizing: border-box; }
          body {
            margin: 0;
            font-family: var(--font-sans);
            font-size: 16px;
            line-height: 1.6;
            color: var(--color-text);
            background: var(--color-bg);
          }
          a { color: var(--color-primary); text-decoration: none; }
          a:hover { color: var(--color-primary-dark); text-decoration: underline; }
          code {
            font-family: var(--font-mono);
            font-size: 0.875em;
            padding: 0.125em 0.375em;
            background: var(--color-bg);
            border-radius: var(--radius-sm);
          }

          .container { max-width: 1200px; margin: 0 auto; padding: 1.5rem; }

          /* Header and Navigation */
          header { margin-bottom: 1.5rem; }
          nav {
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 1rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--color-border);
          }
          nav ul { list-style: none; margin: 0; padding: 0; display: flex; gap: 1rem; align-items: center; }
          nav ul li strong { font-size: 1.25rem; color: var(--color-text); }
          nav ul li a {
            display: inline-block;
            padding: 0.5rem 1rem;
            background: var(--color-primary);
            color: #fff;
            border-radius: var(--radius-sm);
            font-weight: 500;
            font-size: 0.875rem;
            transition: background 0.15s;
          }
          nav ul li a:hover { background: var(--color-primary-dark); text-decoration: none; }

          .back-link { margin-bottom: 1.5rem; }
          .back-link a { display: inline-flex; align-items: center; gap: 0.25rem; font-weight: 500; }

          /* TSL Meta Box */
          .tsl-meta {
            padding: 1rem 1.25rem;
            margin-bottom: 1.5rem;
            border-radius: var(--radius);
            background: var(--color-surface);
            border: 1px solid var(--color-border);
          }
          .tsl-meta p { margin: 0.25rem 0; }
          .tsl-meta p:first-child { margin-top: 0; }
          .tsl-meta p:last-child { margin-bottom: 0; }
          .tsl-meta code { background: var(--color-bg); font-size: 0.8125em; }

          /* Badges */
          .badge {
            display: inline-block;
            padding: 0.2em 0.6em;
            border-radius: var(--radius-sm);
            font-size: 0.75rem;
            font-weight: 600;
            margin-right: 0.5rem;
            margin-bottom: 0.5rem;
            white-space: nowrap;
          }
          .badge-qualified { background: var(--color-success); color: #fff; }
          .badge-nonqualified { background: var(--color-warning); color: #fff; }
          .badge-granted { background: var(--color-success); color: #fff; }
          .badge-withdrawn { background: var(--color-error); color: #fff; }

          /* Details/Accordions */
          details {
            margin-bottom: 1rem;
            background: var(--color-surface);
            border: 1px solid var(--color-border);
            border-radius: var(--radius);
          }
          details summary {
            cursor: pointer;
            padding: 0.75rem 1rem;
            font-weight: 600;
            user-select: none;
            transition: background 0.15s;
          }
          details summary:hover { background: var(--color-bg); }
          details[open] summary { border-bottom: 1px solid var(--color-border); }
          details .content { padding: 1rem; }

          /* Cards */
          article { margin-bottom: 1.5rem; }
          .provider-card {
            background: var(--color-surface);
            border: 1px solid var(--color-border);
            border-radius: var(--radius);
            padding: 1.25rem;
            border-left: 4px solid var(--color-primary);
          }
          .service-card {
            margin-left: 1rem;
            margin-bottom: 1rem;
            padding: 1rem;
            background: var(--color-bg);
            border-radius: var(--radius);
            border-left: 3px solid var(--color-border);
          }

          /* Headings */
          h2 {
            margin: 2rem 0 1rem;
            font-size: 1.5rem;
            font-weight: 600;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid var(--color-primary);
          }
          h3 { margin: 0 0 0.75rem; font-size: 1.25rem; font-weight: 600; }
          h4 { margin: 1rem 0 0.5rem; font-size: 1rem; font-weight: 600; color: var(--color-primary); }
          h5 { margin: 0.75rem 0 0.5rem; font-size: 0.9375rem; font-weight: 600; }

          /* Tables */
          .table-wrapper { overflow-x: auto; margin-bottom: 1rem; }
          table { width: 100%; border-collapse: collapse; font-size: 0.9375rem; }
          table th {
            text-align: left;
            padding: 0.75rem;
            background: var(--color-bg);
            border-bottom: 2px solid var(--color-border);
            font-weight: 600;
            white-space: nowrap;
            width: 160px;
          }
          table td { padding: 0.75rem; border-bottom: 1px solid var(--color-border); vertical-align: top; }
          table tbody tr:last-child td { border-bottom: none; }

          /* URI Display */
          .uri {
            word-break: break-all;
            font-family: var(--font-mono);
            font-size: 0.8125rem;
            color: var(--color-text-muted);
          }

          /* Certificate Data */
          .cert-data {
            font-family: var(--font-mono);
            font-size: 0.75rem;
            max-height: 200px;
            overflow-y: auto;
            padding: 1rem;
            background: #1e1e1e;
            color: #d4d4d4;
            border-radius: var(--radius-sm);
            white-space: pre-wrap;
            word-break: break-all;
            line-height: 1.5;
          }

          /* Footer */
          footer {
            margin-top: 3rem;
            padding-top: 1.5rem;
            border-top: 1px solid var(--color-border);
            text-align: center;
            color: var(--color-text-muted);
            font-size: 0.875rem;
          }

          /* Mobile Responsiveness */
          @media (max-width: 768px) {
            .container { padding: 1rem; }
            nav { flex-direction: column; align-items: flex-start; }
            nav ul { flex-wrap: wrap; }
            .provider-card { padding: 1rem; }
            .service-card { margin-left: 0.5rem; padding: 0.75rem; }
            table { font-size: 0.875rem; }
            table th, table td { padding: 0.5rem; }
            table th { width: 120px; }
            .badge { font-size: 0.6875rem; }
          }

          /* Print Styles */
          @media print {
            nav, .back-link { display: none; }
            body { background: white; }
            details[open] summary { display: block; }
          }
        </style>
      </head>
      <body>
        <main class="container">
          <xsl:apply-templates select="tsl:TrustServiceStatusList"/>
          
          <footer>
            <p><strong>Generated by g119612 TSL Pipeline</strong></p>
          </footer>
        </main>

        <script>
          // Smooth scroll to sections
          document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
              e.preventDefault();
              const target = document.querySelector(this.getAttribute('href'));
              if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
              }
            });
          });
        </script>
      </body>
    </html>
  </xsl:template>
  
  <!-- Process the Trust Service Status List -->
  <xsl:template match="tsl:TrustServiceStatusList">
    <!-- Back to Index Link -->
    <div class="back-link">
      <a href="index.html">← Back to Index</a>
    </div>

    <header>
      <nav>
        <ul>
          <li><strong>
            <xsl:value-of select="tsl:SchemeInformation/tsl:SchemeTerritory"/>
            <xsl:text> Trust Service Status List</xsl:text>
          </strong></li>
        </ul>
        <ul>
          <li><a href="#scheme-info" role="button">Scheme Info</a></li>
          <li><a href="#tsp-list" role="button">Service Providers</a></li>
        </ul>
      </nav>
    </header>
    
    <div class="tsl-meta">
      <p>
        <strong>TSL Sequence #:</strong> <xsl:value-of select="tsl:SchemeInformation/tsl:TSLSequenceNumber"/> | 
        <strong>Issue Date:</strong> <xsl:value-of select="tsl:SchemeInformation/tsl:ListIssueDateTime"/> | 
        <strong>Next Update:</strong> <xsl:value-of select="tsl:SchemeInformation/tsl:NextUpdate/tsl:dateTime"/>
      </p>
      <p>
        <strong>TSL Type:</strong> <code><xsl:value-of select="tsl:SchemeInformation/tsl:TSLType"/></code>
      </p>
    </div>
    
    <article id="scheme-info">
      <h2>Scheme Information</h2>
      <div class="table-wrapper">
        <table>
        <tr>
          <th>Scheme Name</th>
          <td>
            <xsl:for-each select="tsl:SchemeInformation/tsl:SchemeName/tsl:Name">
              <div><xsl:value-of select="."/> (<xsl:value-of select="@xml:lang"/>)</div>
            </xsl:for-each>
          </td>
        </tr>
        <tr>
          <th>Scheme Operator</th>
          <td>
            <xsl:for-each select="tsl:SchemeInformation/tsl:SchemeOperatorName/tsl:Name">
              <div><xsl:value-of select="."/> (<xsl:value-of select="@xml:lang"/>)</div>
            </xsl:for-each>
          </td>
        </tr>
        <tr>
          <th>Status Determination</th>
          <td><xsl:value-of select="tsl:SchemeInformation/tsl:StatusDeterminationApproach"/></td>
        </tr>
        <tr>
          <th>Scheme Territory</th>
          <td><xsl:value-of select="tsl:SchemeInformation/tsl:SchemeTerritory"/></td>
        </tr>
        <tr>
          <th>Historical Information Period</th>
          <td><xsl:value-of select="tsl:SchemeInformation/tsl:HistoricalInformationPeriod"/> days</td>
        </tr>
        <tr>
          <th>Scheme URLs</th>
          <td>
            <xsl:for-each select="tsl:SchemeInformation/tsl:SchemeInformationURI/tsl:URI">
              <div class="uri"><xsl:value-of select="."/></div>
            </xsl:for-each>
          </td>
        </tr>
        <tr>
          <th>Distribution Points</th>
          <td>
            <xsl:for-each select="tsl:SchemeInformation/tsl:DistributionPoints/tsl:URI">
              <div class="uri"><xsl:value-of select="."/></div>
            </xsl:for-each>
          </td>
        </tr>
      </table>
      </div>
      
      <details>
        <summary>Policy/Legal Notice</summary>
        <div class="content">
          <xsl:for-each select="tsl:SchemeInformation/tsl:PolicyOrLegalNotice/tsl:TSLLegalNotice">
            <p><strong>Language:</strong> <xsl:value-of select="@xml:lang"/></p>
            <p><xsl:value-of select="."/></p>
          </xsl:for-each>
        </div>
      </details>
      
      <h3>Pointers to Other TSLs</h3>
      <xsl:choose>
        <xsl:when test="tsl:SchemeInformation/tsl:PointersToOtherTSL/tsl:OtherTSLPointer">
          <div class="table-wrapper">
            <table>
            <thead>
              <tr>
                <th>TSL Type</th>
                <th>Territory</th>
                <th>Scheme Name</th>
                <th>URL</th>
              </tr>
            </thead>
            <tbody>
              <xsl:for-each select="tsl:SchemeInformation/tsl:PointersToOtherTSL/tsl:OtherTSLPointer">
                <tr>
                  <td><xsl:value-of select="tsl:TSLType"/></td>
                  <td><xsl:value-of select="tsl:SchemeTerritory"/></td>
                  <td>
                    <xsl:for-each select="tsl:SchemeOperatorName/tsl:Name[1]">
                      <xsl:value-of select="."/>
                    </xsl:for-each>
                  </td>
                  <td class="uri"><xsl:value-of select="tsl:TSLLocation"/></td>
                </tr>
              </xsl:for-each>
            </tbody>
          </table>
          </div>
        </xsl:when>
        <xsl:otherwise>
          <p>No pointers to other TSLs found.</p>
        </xsl:otherwise>
      </xsl:choose>
    </article>
    
    <article id="tsp-list">
      <h2>Trust Service Providers</h2>
      <xsl:choose>
        <xsl:when test="tsl:TrustServiceProviderList/tsl:TrustServiceProvider">
          <xsl:apply-templates select="tsl:TrustServiceProviderList/tsl:TrustServiceProvider"/>
        </xsl:when>
        <xsl:otherwise>
          <article>
            <p>No trust service providers found in this TSL.</p>
          </article>
        </xsl:otherwise>
      </xsl:choose>
    </article>
  </xsl:template>
  
  <!-- Process each Trust Service Provider -->
  <xsl:template match="tsl:TrustServiceProvider">
    <article class="provider-card">
      <h3>
        <xsl:value-of select="tsl:TSPInformation/tsl:TSPName/tsl:Name[1]"/>
      </h3>
      
      <h4>Provider Information</h4>
      <div class="table-wrapper">
        <table>
        <tr>
          <th>TSP Name</th>
          <td>
            <xsl:for-each select="tsl:TSPInformation/tsl:TSPName/tsl:Name">
              <div><xsl:value-of select="."/> (<xsl:value-of select="@xml:lang"/>)</div>
            </xsl:for-each>
          </td>
        </tr>
        <xsl:if test="tsl:TSPInformation/tsl:TSPTradeName">
          <tr>
            <th>Trade Name</th>
            <td>
              <xsl:for-each select="tsl:TSPInformation/tsl:TSPTradeName/tsl:Name">
                <div><xsl:value-of select="."/> (<xsl:value-of select="@xml:lang"/>)</div>
              </xsl:for-each>
            </td>
          </tr>
        </xsl:if>
        <tr>
          <th>Information URLs</th>
          <td>
            <xsl:for-each select="tsl:TSPInformation/tsl:TSPInformationURI/tsl:URI">
              <div class="uri"><xsl:value-of select="."/> (<xsl:value-of select="@xml:lang"/>)</div>
            </xsl:for-each>
          </td>
        </tr>
      </table>
      </div>
      
      <details>
        <summary>Contact Details</summary>
        <div class="content">
          <h5>Address</h5>
          <xsl:for-each select="tsl:TSPInformation/tsl:TSPAddress/tsl:PostalAddresses/tsl:PostalAddress">
            <p>
              <strong>Language:</strong> <xsl:value-of select="@xml:lang"/><br/>
              <strong>Street:</strong> <xsl:value-of select="tsl:StreetAddress"/><br/>
              <strong>Locality:</strong> <xsl:value-of select="tsl:Locality"/><br/>
              <strong>Postal Code:</strong> <xsl:value-of select="tsl:PostalCode"/><br/>
              <strong>Country:</strong> <xsl:value-of select="tsl:CountryName"/>
            </p>
          </xsl:for-each>
          
          <h5>Electronic Address</h5>
          <xsl:for-each select="tsl:TSPInformation/tsl:TSPAddress/tsl:ElectronicAddress/tsl:URI">
            <p><a href="{.}"><xsl:value-of select="."/></a></p>
          </xsl:for-each>
        </div>
      </details>
      
      <h4>Services</h4>
      <xsl:apply-templates select="tsl:TSPServices/tsl:TSPService"/>
    </article>
  </xsl:template>
  
  <!-- Process each Trust Service -->
  <xsl:template match="tsl:TSPService">
    <article class="service-card">
      <xsl:variable name="serviceType" select="tsl:ServiceInformation/tsl:ServiceTypeIdentifier"/>
      <xsl:variable name="currentStatus" select="tsl:ServiceInformation/tsl:ServiceStatus"/>
      
      <h4>
        <xsl:value-of select="tsl:ServiceInformation/tsl:ServiceName/tsl:Name[1]"/>
      </h4>
      
      <div>
        <!-- Service Type Badge -->
        <xsl:choose>
          <xsl:when test="contains($serviceType, '/QC')">
            <span class="badge badge-qualified">Qualified</span>
          </xsl:when>
          <xsl:otherwise>
            <span class="badge badge-nonqualified">Non-Qualified</span>
          </xsl:otherwise>
        </xsl:choose>
        
        <!-- Service Status Badge -->
        <xsl:choose>
          <xsl:when test="contains($currentStatus, 'granted')">
            <span class="badge badge-granted">Granted</span>
          </xsl:when>
          <xsl:when test="contains($currentStatus, 'withdrawn')">
            <span class="badge badge-withdrawn">Withdrawn</span>
          </xsl:when>
          <xsl:otherwise>
            <span class="badge"><xsl:value-of select="substring-after($currentStatus, 'StatusDetn/')"/></span>
          </xsl:otherwise>
        </xsl:choose>
      </div>
      
      <div class="table-wrapper">
        <table>
        <tr>
          <th>Service Type</th>
          <td class="uri"><code><xsl:value-of select="$serviceType"/></code></td>
        </tr>
        <tr>
          <th>Status</th>
          <td class="uri"><code><xsl:value-of select="$currentStatus"/></code></td>
        </tr>
        <tr>
          <th>Status Starting Time</th>
          <td><xsl:value-of select="tsl:ServiceInformation/tsl:StatusStartingTime"/></td>
        </tr>
      </table>
      </div>
      
      <details>
        <summary>Service Digital Identity</summary>
        <div class="content">
          <xsl:for-each select="tsl:ServiceInformation/tsl:ServiceDigitalIdentity/tsl:DigitalId/ns2:X509Certificate">
            <h5>Certificate</h5>
            <div class="cert-data"><xsl:value-of select="."/></div>
          </xsl:for-each>
          
          <xsl:for-each select="tsl:ServiceInformation/tsl:ServiceDigitalIdentity/tsl:DigitalId/*[local-name() != 'X509Certificate']">
            <h5><xsl:value-of select="local-name()"/></h5>
            <div class="cert-data"><xsl:value-of select="."/></div>
          </xsl:for-each>
        </div>
      </details>
      
      <!-- Service Information Extensions -->
      <xsl:if test="tsl:ServiceInformation/tsl:ServiceInformationExtensions">
        <details>
          <summary>Service Extensions</summary>
          <div class="content">
            <xsl:for-each select="tsl:ServiceInformation/tsl:ServiceInformationExtensions/*">
              <h5><xsl:value-of select="local-name()"/></h5>
              <div>
                <xsl:choose>
                  <xsl:when test="@*">
                    <table>
                      <xsl:for-each select="@*">
                        <tr>
                          <th><xsl:value-of select="name()"/></th>
                          <td><xsl:value-of select="."/></td>
                        </tr>
                      </xsl:for-each>
                    </table>
                  </xsl:when>
                  <xsl:otherwise>
                    <xsl:value-of select="."/>
                  </xsl:otherwise>
                </xsl:choose>
              </div>
            </xsl:for-each>
          </div>
        </details>
      </xsl:if>
      
      <!-- Service History -->
      <xsl:if test="tsl:ServiceHistory">
        <details>
          <summary>Service History</summary>
          <div class="content">
            <h5>Historical Service Information</h5>
            <xsl:for-each select="tsl:ServiceHistory/tsl:ServiceHistoryInstance">
              <article style="margin-bottom: 15px; padding-bottom: 15px; border-bottom: 1px solid var(--card-border-color);">
                <p>
                  <strong>Service Type:</strong> <code><xsl:value-of select="tsl:ServiceTypeIdentifier"/></code><br/>
                  <strong>Service Name:</strong> <xsl:value-of select="tsl:ServiceName/tsl:Name[1]"/><br/>
                  <strong>Status:</strong> <code><xsl:value-of select="tsl:ServiceStatus"/></code><br/>
                  <strong>Status Starting Time:</strong> <xsl:value-of select="tsl:StatusStartingTime"/>
                </p>
              </article>
            </xsl:for-each>
          </div>
        </details>
      </xsl:if>
    </article>
  </xsl:template>
</xsl:stylesheet>