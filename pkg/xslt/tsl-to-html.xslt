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
    <html lang="en" data-theme="light">
      <head>
        <meta charset="UTF-8"/>
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
        <title>
          <xsl:value-of select="tsl:TrustServiceStatusList/tsl:SchemeInformation/tsl:SchemeTerritory"/>
          <xsl:text> - Trust Service Status List</xsl:text>
        </title>
        <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css"/>
        <style>
          /* Custom styles to complement PicoCSS */
          :root {
            --badge-qualified-bg: #27ae60;
            --badge-nonqualified-bg: #f39c12;
            --badge-granted-bg: #2ecc71;
            --badge-withdrawn-bg: #e74c3c;
          }
          
          body {
            padding-bottom: 2rem;
          }

          .container {
            max-width: 1400px;
          }

          /* Header Improvements */
          nav {
            margin-bottom: 1.5rem;
          }

          nav ul li strong {
            font-size: 1.2rem;
          }

          /* Back to Index Button */
          .back-link {
            margin-bottom: 1rem;
          }

          .back-link a {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            background: var(--primary);
            color: white;
            border-radius: 5px;
            text-decoration: none;
            font-weight: 600;
          }

          .back-link a:hover {
            opacity: 0.9;
          }

          /* Theme Toggle */
          .theme-toggle {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            padding: 0.75rem;
            background: var(--primary);
            color: white;
            border: none;
            border-radius: 50%;
            cursor: pointer;
            z-index: 1000;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            width: 50px;
            height: 50px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
          }

          .theme-toggle:hover {
            opacity: 0.9;
            transform: scale(1.05);
          }

          /* TSL Meta Box */
          .tsl-meta {
            padding: 1.25rem;
            margin-bottom: 1.5rem;
            border-radius: 8px;
            background-color: var(--card-background-color);
            border: 1px solid var(--card-border-color);
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
          }

          .tsl-meta p {
            margin-bottom: 0.5rem;
          }

          .tsl-meta p:last-child {
            margin-bottom: 0;
          }

          /* Certificate Data */
          .cert-data {
            font-family: 'Courier New', Courier, monospace;
            font-size: 0.75rem;
            max-height: 200px;
            overflow-y: auto;
            padding: 1rem;
            border: 1px solid var(--card-border-color);
            border-radius: 5px;
            background-color: var(--code-background-color);
            white-space: pre-wrap;
            word-break: break-all;
            line-height: 1.4;
          }

          /* Badges */
          .badge {
            display: inline-block;
            padding: 0.3rem 0.7rem;
            border-radius: 4px;
            font-size: 0.8rem;
            font-weight: 600;
            margin-right: 0.5rem;
            margin-bottom: 0.5rem;
            white-space: nowrap;
          }
          
          .badge-qualified {
            background-color: var(--badge-qualified-bg);
            color: white;
          }
          
          .badge-nonqualified {
            background-color: var(--badge-nonqualified-bg);
            color: white;
          }
          
          .badge-granted {
            background-color: var(--badge-granted-bg);
            color: white;
          }
          
          .badge-withdrawn {
            background-color: var(--badge-withdrawn-bg);
            color: white;
          }

          /* Details/Summary Improvements */
          details {
            margin-bottom: 1rem;
          }
          
          details summary {
            cursor: pointer;
            padding: 0.75rem 1rem;
            background-color: var(--card-background-color);
            border: 1px solid var(--card-border-color);
            border-radius: 5px;
            font-weight: 600;
            transition: background-color 0.2s;
            user-select: none;
          }

          details summary:hover {
            background-color: var(--primary-hover);
          }
          
          details[open] summary {
            border-bottom-left-radius: 0;
            border-bottom-right-radius: 0;
            margin-bottom: 0;
            background-color: var(--primary-hover);
          }
          
          details .content {
            padding: 1rem;
            border: 1px solid var(--card-border-color);
            border-top: none;
            border-bottom-left-radius: 5px;
            border-bottom-right-radius: 5px;
            background-color: var(--card-background-color);
          }

          /* Service Cards */
          .service-card {
            margin-left: 1.5rem;
            margin-bottom: 1.5rem;
            padding-left: 1rem;
            border-left: 4px solid var(--primary-focus);
          }

          /* Provider Cards */
          .provider-card {
            border-left: 4px solid var(--primary);
            padding-left: 1rem;
            margin-bottom: 2rem;
          }

          /* URI Display */
          .uri {
            word-break: break-all;
            font-family: 'Courier New', Courier, monospace;
            font-size: 0.85em;
            line-height: 1.4;
          }

          /* Articles */
          article {
            margin-bottom: 2rem;
          }

          /* Tables - Responsive */
          .table-wrapper {
            overflow-x: auto;
            margin-bottom: 1rem;
          }

          table {
            width: 100%;
            min-width: auto;
          }

          table th {
            white-space: nowrap;
            background-color: var(--card-background-color);
            padding: 0.75rem;
          }

          table td {
            padding: 0.75rem;
            vertical-align: top;
          }

          /* Headings */
          h2 {
            margin-top: 2rem;
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid var(--primary);
          }

          h3 {
            margin-top: 1.5rem;
            margin-bottom: 0.75rem;
          }

          h4 {
            margin-top: 1rem;
            margin-bottom: 0.5rem;
            color: var(--primary);
          }

          /* Footer */
          footer {
            margin-top: 3rem;
            padding-top: 2rem;
            border-top: 1px solid var(--card-border-color);
            text-align: center;
            color: var(--muted-color);
            font-size: 0.9rem;
          }
          
          /* Dark mode compatibility */
          @media (prefers-color-scheme: dark) {
            :root:not([data-theme="light"]) {
              --badge-qualified-bg: #27ae60;
              --badge-nonqualified-bg: #f39c12;
              --badge-granted-bg: #2ecc71;
              --badge-withdrawn-bg: #e74c3c;
            }

            .cert-data {
              background-color: #1a1a1a;
            }
          }

          /* Mobile Responsiveness */
          @media (max-width: 768px) {
            .container {
              padding: 1rem;
            }

            nav ul li strong {
              font-size: 1rem;
            }

            .service-card {
              margin-left: 0.5rem;
              padding-left: 0.75rem;
            }

            .provider-card {
              padding-left: 0.75rem;
            }

            table {
              font-size: 0.85rem;
            }

            table th,
            table td {
              padding: 0.5rem;
            }

            .badge {
              font-size: 0.7rem;
              padding: 0.25rem 0.5rem;
            }

            .cert-data {
              font-size: 0.7rem;
              padding: 0.75rem;
              max-height: 150px;
            }

            h2 {
              font-size: 1.5rem;
            }

            h3 {
              font-size: 1.25rem;
            }

            h4 {
              font-size: 1.1rem;
            }

            .theme-toggle {
              bottom: 1rem;
              right: 1rem;
              width: 45px;
              height: 45px;
              font-size: 1.25rem;
            }

            /* Stack table rows vertically on very small screens */
            @media (max-width: 480px) {
              table {
                font-size: 0.8rem;
              }

              table th {
                min-width: 100px;
              }
            }
          }

          /* Print Styles */
          @media print {
            .theme-toggle,
            nav,
            .back-link {
              display: none;
            }

            body {
              background: white;
            }

            details {
              page-break-inside: avoid;
            }

            details summary {
              display: none;
            }

            details .content {
              border: none;
              padding: 0;
            }
          }
        </style>
      </head>
      <body>
        <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle dark mode">🌓</button>
        
        <main class="container">
          <xsl:apply-templates select="tsl:TrustServiceStatusList"/>
          
          <footer>
            <p><strong>Generated using TSL to HTML Stylesheet</strong><br/>
            Styled with PicoCSS</p>
          </footer>
        </main>

        <script>
          // Theme toggle functionality
          function toggleTheme() {
            const html = document.documentElement;
            const currentTheme = html.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            html.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
          }

          // Load saved theme
          document.addEventListener('DOMContentLoaded', function() {
            const savedTheme = localStorage.getItem('theme') || 
              (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
            document.documentElement.setAttribute('data-theme', savedTheme);
          });

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