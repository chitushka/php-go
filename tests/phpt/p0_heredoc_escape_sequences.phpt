--TEST--
P0 heredoc processes escape sequences like double-quoted strings
--FILE--
<?php
$name = 'PHP';

$heredoc = <<<TXT
Tab:\tNewline:\nBackslash:\\Dollar:\$name Hex:\x41
TXT;

$nowdoc = <<<'RAW'
Raw:\tRaw:\\Raw:$name
RAW;

echo $heredoc;
echo "\n---\n";
echo $nowdoc;
?>
--EXPECT--
Tab:	Newline:
Backslash:\Dollar:$name Hex:A
---
Raw:\tRaw:\\Raw:$name
