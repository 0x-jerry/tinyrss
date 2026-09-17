package feeds

const rssXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel>
<title>Example Blog</title>
<link>https://example.com/</link>
<description>A demo blog</description>
<item>
<title>Hello World</title>
<link>https://example.com/post/1</link>
<guid>guid-1</guid>
<pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
<author>alice@example.com (Alice)</author>
<description>Short summary text</description>
<content:encoded><![CDATA[Full <b>content</b> body]]></content:encoded>
</item>
</channel>
</rss>`

const atomXML = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Example Atom</title>
<link href="https://example.com/atom/"/>
<updated>2003-12-13T18:30:02Z</updated>
<entry>
<title>Atom Entry</title>
<link href="https://example.com/atom/1"/>
<id>urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a</id>
<updated>2003-12-13T18:30:02Z</updated>
<summary>Atom summary text</summary>
<content type="html">Atom &lt;b&gt;bold&lt;/b&gt; content</content>
<author><name>Bob</name></author>
</entry>
</feed>`
