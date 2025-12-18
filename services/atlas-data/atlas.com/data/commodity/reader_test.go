package commodity

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
	"strconv"
	"testing"
)

const testXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="Commodity.img">
  <imgdir name="0">
    <int name="SN" value="10000000"/>
    <int name="ItemId" value="1002000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="1">
    <int name="SN" value="10000001"/>
    <int name="ItemId" value="1002015"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="2">
    <int name="SN" value="10000002"/>
    <int name="ItemId" value="1002187"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="3">
    <int name="SN" value="10000003"/>
    <int name="ItemId" value="1002292"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="4">
    <int name="SN" value="10000004"/>
    <int name="ItemId" value="1042018"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="5">
    <int name="SN" value="10000005"/>
    <int name="ItemId" value="1042024"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="6">
    <int name="SN" value="10000006"/>
    <int name="ItemId" value="1050050"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="7">
    <int name="SN" value="10000007"/>
    <int name="ItemId" value="1022001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="8">
    <int name="SN" value="10000008"/>
    <int name="ItemId" value="1022015"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="9">
    <int name="SN" value="10000009"/>
    <int name="ItemId" value="1072111"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="10">
    <int name="SN" value="10000010"/>
    <int name="ItemId" value="1001002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="11">
    <int name="SN" value="10000011"/>
    <int name="ItemId" value="1002224"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="12">
    <int name="SN" value="10000012"/>
    <int name="ItemId" value="1002314"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="13">
    <int name="SN" value="10000013"/>
    <int name="ItemId" value="1002312"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="14">
    <int name="SN" value="10000014"/>
    <int name="ItemId" value="1002031"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="15">
    <int name="SN" value="10000015"/>
    <int name="ItemId" value="1002470"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="16">
    <int name="SN" value="10000016"/>
    <int name="ItemId" value="1051002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="17">
    <int name="SN" value="10000017"/>
    <int name="ItemId" value="1050017"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="18">
    <int name="SN" value="10000018"/>
    <int name="ItemId" value="1051048"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="19">
    <int name="SN" value="10000019"/>
    <int name="ItemId" value="1022029"/>
    <int name="Count" value="1"/>
    <int name="Price" value="300"/>
    <int name="Period" value="14"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="20">
    <int name="SN" value="10000020"/>
    <int name="ItemId" value="1002190"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="21">
    <int name="SN" value="10000021"/>
    <int name="ItemId" value="1002191"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="22">
    <int name="SN" value="10000022"/>
    <int name="ItemId" value="1022004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="23">
    <int name="SN" value="10000023"/>
    <int name="ItemId" value="1052025"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4600"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="24">
    <int name="SN" value="10000024"/>
    <int name="ItemId" value="1040078"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="25">
    <int name="SN" value="10000025"/>
    <int name="ItemId" value="1041073"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="26">
    <int name="SN" value="10000026"/>
    <int name="ItemId" value="1060067"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="27">
    <int name="SN" value="10000027"/>
    <int name="ItemId" value="1062003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="28">
    <int name="SN" value="10000028"/>
    <int name="ItemId" value="1061068"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="29">
    <int name="SN" value="10000029"/>
    <int name="ItemId" value="1072189"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="30">
    <int name="SN" value="10000030"/>
    <int name="ItemId" value="1002469"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="31">
    <int name="SN" value="10000031"/>
    <int name="ItemId" value="1012029"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="32">
    <int name="SN" value="10000032"/>
    <int name="ItemId" value="1002186"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="33">
    <int name="SN" value="10000033"/>
    <int name="ItemId" value="5150000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="34">
    <int name="SN" value="10000034"/>
    <int name="ItemId" value="5151000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="35">
    <int name="SN" value="10000035"/>
    <int name="ItemId" value="5120008"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="36">
    <int name="SN" value="10000036"/>
    <int name="ItemId" value="1002226"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="37">
    <int name="SN" value="10000037"/>
    <int name="ItemId" value="1041000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="38">
    <int name="SN" value="10000038"/>
    <int name="ItemId" value="1002232"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="39">
    <int name="SN" value="10000039"/>
    <int name="ItemId" value="1042033"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="40">
    <int name="SN" value="10000040"/>
    <int name="ItemId" value="1022008"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="41">
    <int name="SN" value="10000041"/>
    <int name="ItemId" value="1052029"/>
    <int name="Count" value="1"/>
    <int name="Price" value="5100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="42">
    <int name="SN" value="10000042"/>
    <int name="ItemId" value="1002203"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="43">
    <int name="SN" value="10000043"/>
    <int name="ItemId" value="1002235"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="44">
    <int name="SN" value="10000044"/>
    <int name="ItemId" value="1002250"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="45">
    <int name="SN" value="10000045"/>
    <int name="ItemId" value="1042017"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="46">
    <int name="SN" value="10000046"/>
    <int name="ItemId" value="1042021"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="47">
    <int name="SN" value="10000047"/>
    <int name="ItemId" value="1042013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="48">
    <int name="SN" value="10000048"/>
    <int name="ItemId" value="1052005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="49">
    <int name="SN" value="10000049"/>
    <int name="ItemId" value="1050032"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="50">
    <int name="SN" value="10000050"/>
    <int name="ItemId" value="1022020"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="51">
    <int name="SN" value="10000051"/>
    <int name="ItemId" value="1022025"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="52">
    <int name="SN" value="10000052"/>
    <int name="ItemId" value="1002343"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="53">
    <int name="SN" value="10000053"/>
    <int name="ItemId" value="1002367"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="54">
    <int name="SN" value="10000054"/>
    <int name="ItemId" value="1002368"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="55">
    <int name="SN" value="10000055"/>
    <int name="ItemId" value="1042036"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="56">
    <int name="SN" value="10000056"/>
    <int name="ItemId" value="1052030"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="57">
    <int name="SN" value="10000057"/>
    <int name="ItemId" value="1051049"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="58">
    <int name="SN" value="10000058"/>
    <int name="ItemId" value="1012007"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="59">
    <int name="SN" value="10000059"/>
    <int name="ItemId" value="1702004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="60">
    <int name="SN" value="10000060"/>
    <int name="ItemId" value="1702008"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="61">
    <int name="SN" value="10000061"/>
    <int name="ItemId" value="5120004"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="62">
    <int name="SN" value="10000062"/>
    <int name="ItemId" value="1102005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="63">
    <int name="SN" value="10000063"/>
    <int name="ItemId" value="5000000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="64">
    <int name="SN" value="10000064"/>
    <int name="ItemId" value="5000001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="65">
    <int name="SN" value="10000065"/>
    <int name="ItemId" value="5000014"/>
    <int name="Count" value="1"/>
    <int name="Price" value="6700"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="66">
    <int name="SN" value="10000066"/>
    <int name="ItemId" value="1812000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="67">
    <int name="SN" value="10000067"/>
    <int name="ItemId" value="1002266"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="68">
    <int name="SN" value="10000068"/>
    <int name="ItemId" value="1072217"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="69">
    <int name="SN" value="10000069"/>
    <int name="ItemId" value="1112109"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="70">
    <int name="SN" value="10000070"/>
    <int name="ItemId" value="1002352"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="71">
    <int name="SN" value="10000071"/>
    <int name="ItemId" value="1002356"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="72">
    <int name="SN" value="10000072"/>
    <int name="ItemId" value="1002228"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="73">
    <int name="SN" value="10000073"/>
    <int name="ItemId" value="1012005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="74">
    <int name="SN" value="10000074"/>
    <int name="ItemId" value="1022024"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="75">
    <int name="SN" value="10000075"/>
    <int name="ItemId" value="1072010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="76">
    <int name="SN" value="10000076"/>
    <int name="ItemId" value="1802000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="77">
    <int name="SN" value="10000077"/>
    <int name="ItemId" value="1112212"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="78">
    <int name="SN" value="10000078"/>
    <int name="ItemId" value="5150004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="79">
    <int name="SN" value="10000079"/>
    <int name="ItemId" value="5151004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="80">
    <int name="SN" value="10000080"/>
    <int name="ItemId" value="1022023"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="81">
    <int name="SN" value="10000081"/>
    <int name="ItemId" value="1042051"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="82">
    <int name="SN" value="10000082"/>
    <int name="ItemId" value="1042053"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="83">
    <int name="SN" value="10000083"/>
    <int name="ItemId" value="1062045"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="84">
    <int name="SN" value="10000084"/>
    <int name="ItemId" value="1812001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="85">
    <int name="SN" value="10000085"/>
    <int name="ItemId" value="1112108"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="86">
    <int name="SN" value="10000086"/>
    <int name="ItemId" value="1702010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="87">
    <int name="SN" value="10000087"/>
    <int name="ItemId" value="5120002"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="88">
    <int name="SN" value="10000088"/>
    <int name="ItemId" value="5152004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="89">
    <int name="SN" value="10000089"/>
    <int name="ItemId" value="5160002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="90">
    <int name="SN" value="10000090"/>
    <int name="ItemId" value="1112209"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="91">
    <int name="SN" value="10000091"/>
    <int name="ItemId" value="5160004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="92">
    <int name="SN" value="10000092"/>
    <int name="ItemId" value="5160005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="93">
    <int name="SN" value="10000093"/>
    <int name="ItemId" value="5000003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="5100"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="94">
    <int name="SN" value="10000094"/>
    <int name="ItemId" value="1802014"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="95">
    <int name="SN" value="10000095"/>
    <int name="ItemId" value="5153001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="96">
    <int name="SN" value="10000096"/>
    <int name="ItemId" value="1082102"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="97">
    <int name="SN" value="10000097"/>
    <int name="ItemId" value="1702020"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="98">
    <int name="SN" value="10000098"/>
    <int name="ItemId" value="1702024"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="99">
    <int name="SN" value="10000099"/>
    <int name="ItemId" value="1702025"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="100">
    <int name="SN" value="10000100"/>
    <int name="ItemId" value="1702026"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="101">
    <int name="SN" value="10000101"/>
    <int name="ItemId" value="1112001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="6000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="102">
    <int name="SN" value="10000102"/>
    <int name="ItemId" value="5120005"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="103">
    <int name="SN" value="10000104"/>
    <int name="ItemId" value="5120007"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="104">
    <int name="SN" value="10000106"/>
    <int name="ItemId" value="5140000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="105">
    <int name="SN" value="10000107"/>
    <int name="ItemId" value="5160009"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="106">
    <int name="SN" value="10000108"/>
    <int name="ItemId" value="1802006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="107">
    <int name="SN" value="10000109"/>
    <int name="ItemId" value="1802007"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="108">
    <int name="SN" value="10000110"/>
    <int name="ItemId" value="1002421"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="109">
    <int name="SN" value="10000111"/>
    <int name="ItemId" value="1002422"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="110">
    <int name="SN" value="10000112"/>
    <int name="ItemId" value="1002423"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="111">
    <int name="SN" value="10000113"/>
    <int name="ItemId" value="1012028"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="112">
    <int name="SN" value="10000114"/>
    <int name="ItemId" value="1041125"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="113">
    <int name="SN" value="10000115"/>
    <int name="ItemId" value="1041134"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="114">
    <int name="SN" value="10000116"/>
    <int name="ItemId" value="1060114"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="115">
    <int name="SN" value="10000117"/>
    <int name="ItemId" value="1072153"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="116">
    <int name="SN" value="10000118"/>
    <int name="ItemId" value="1702017"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="117">
    <int name="SN" value="10000119"/>
    <int name="ItemId" value="1702013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="118">
    <int name="SN" value="10000120"/>
    <int name="ItemId" value="1112210"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="119">
    <int name="SN" value="10000121"/>
    <int name="ItemId" value="5050000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="120">
    <int name="SN" value="10000122"/>
    <int name="ItemId" value="1002299"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="121">
    <int name="SN" value="10000123"/>
    <int name="ItemId" value="1040124"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="122">
    <int name="SN" value="10000124"/>
    <int name="ItemId" value="1040125"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="123">
    <int name="SN" value="10000125"/>
    <int name="ItemId" value="1062038"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="124">
    <int name="SN" value="10000126"/>
    <int name="ItemId" value="1102039"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="125">
    <int name="SN" value="10000127"/>
    <int name="ItemId" value="5050001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="126">
    <int name="SN" value="10000128"/>
    <int name="ItemId" value="1012000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="127">
    <int name="SN" value="10000129"/>
    <int name="ItemId" value="1042038"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="128">
    <int name="SN" value="10000130"/>
    <int name="ItemId" value="1102007"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="129">
    <int name="SN" value="10000131"/>
    <int name="ItemId" value="1102008"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="130">
    <int name="SN" value="10000132"/>
    <int name="ItemId" value="5150006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="131">
    <int name="SN" value="10000133"/>
    <int name="ItemId" value="5151006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="132">
    <int name="SN" value="10000134"/>
    <int name="ItemId" value="1002428"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="133">
    <int name="SN" value="10000135"/>
    <int name="ItemId" value="1001013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="134">
    <int name="SN" value="10000136"/>
    <int name="ItemId" value="1072013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="135">
    <int name="SN" value="10000137"/>
    <int name="ItemId" value="1072057"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="136">
    <int name="SN" value="10000138"/>
    <int name="ItemId" value="1102009"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="137">
    <int name="SN" value="10000139"/>
    <int name="ItemId" value="1102010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="138">
    <int name="SN" value="10000140"/>
    <int name="ItemId" value="5000004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="139">
    <int name="SN" value="10000141"/>
    <int name="ItemId" value="1802001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="140">
    <int name="SN" value="10000142"/>
    <int name="ItemId" value="1072058"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="141">
    <int name="SN" value="10000143"/>
    <int name="ItemId" value="5152006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="142">
    <int name="SN" value="10000144"/>
    <int name="ItemId" value="5150000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="143">
    <int name="SN" value="10000145"/>
    <int name="ItemId" value="5160001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="144">
    <int name="SN" value="10000146"/>
    <int name="ItemId" value="1802002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="145">
    <int name="SN" value="10000147"/>
    <int name="ItemId" value="1802003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="146">
    <int name="SN" value="10000148"/>
    <int name="ItemId" value="1002429"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="147">
    <int name="SN" value="10000149"/>
    <int name="ItemId" value="1050111"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="148">
    <int name="SN" value="10000150"/>
    <int name="ItemId" value="1051112"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="149">
    <int name="SN" value="10000151"/>
    <int name="ItemId" value="1072230"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="150">
    <int name="SN" value="10000152"/>
    <int name="ItemId" value="1072231"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="151">
    <int name="SN" value="10000153"/>
    <int name="ItemId" value="1072232"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="152">
    <int name="SN" value="10000154"/>
    <int name="ItemId" value="5130000"/>
    <int name="Count" value="5"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="153">
    <int name="SN" value="10000155"/>
    <int name="ItemId" value="1001018"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="154">
    <int name="SN" value="10000156"/>
    <int name="ItemId" value="1001019"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="155">
    <int name="SN" value="10000157"/>
    <int name="ItemId" value="1001020"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="156">
    <int name="SN" value="10000158"/>
    <int name="ItemId" value="1011003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="157">
    <int name="SN" value="10000159"/>
    <int name="ItemId" value="1042039"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="158">
    <int name="SN" value="10000160"/>
    <int name="ItemId" value="1042040"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="159">
    <int name="SN" value="10000161"/>
    <int name="ItemId" value="1042041"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="160">
    <int name="SN" value="10000162"/>
    <int name="ItemId" value="1061127"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="161">
    <int name="SN" value="10000163"/>
    <int name="ItemId" value="1061128"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="162">
    <int name="SN" value="10000164"/>
    <int name="ItemId" value="1702010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="163">
    <int name="SN" value="10000165"/>
    <int name="ItemId" value="1702011"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="164">
    <int name="SN" value="10000166"/>
    <int name="ItemId" value="1112213"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="165">
    <int name="SN" value="10000167"/>
    <int name="ItemId" value="5150002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="166">
    <int name="SN" value="10000168"/>
    <int name="ItemId" value="1001013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="167">
    <int name="SN" value="10000169"/>
    <int name="ItemId" value="1072013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="168">
    <int name="SN" value="10000170"/>
    <int name="ItemId" value="1072057"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="169">
    <int name="SN" value="10000171"/>
    <int name="ItemId" value="1102009"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="170">
    <int name="SN" value="10000172"/>
    <int name="ItemId" value="1102010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="171">
    <int name="SN" value="10000173"/>
    <int name="ItemId" value="5000004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="5100"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="172">
    <int name="SN" value="10000174"/>
    <int name="ItemId" value="1802001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="173">
    <int name="SN" value="10000175"/>
    <int name="ItemId" value="1072058"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="174">
    <int name="SN" value="10000176"/>
    <int name="ItemId" value="5152006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="175">
    <int name="SN" value="10000177"/>
    <int name="ItemId" value="5160000"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="176">
    <int name="SN" value="10000178"/>
    <int name="ItemId" value="5160001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="177">
    <int name="SN" value="10000179"/>
    <int name="ItemId" value="1802002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="178">
    <int name="SN" value="10000180"/>
    <int name="ItemId" value="1802003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="179">
    <int name="SN" value="10000181"/>
    <int name="ItemId" value="1002429"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="180">
    <int name="SN" value="10000182"/>
    <int name="ItemId" value="1050111"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="0"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="181">
    <int name="SN" value="10000183"/>
    <int name="ItemId" value="1051112"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="182">
    <int name="SN" value="10000184"/>
    <int name="ItemId" value="1072230"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="183">
    <int name="SN" value="10000185"/>
    <int name="ItemId" value="1072231"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="184">
    <int name="SN" value="10000186"/>
    <int name="ItemId" value="1072232"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="185">
    <int name="SN" value="10000187"/>
    <int name="ItemId" value="5130000"/>
    <int name="Count" value="5"/>
    <int name="Price" value="3200"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="186">
    <int name="SN" value="10000188"/>
    <int name="ItemId" value="1001018"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="187">
    <int name="SN" value="10000189"/>
    <int name="ItemId" value="1001019"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="188">
    <int name="SN" value="10000190"/>
    <int name="ItemId" value="1001020"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="189">
    <int name="SN" value="10000191"/>
    <int name="ItemId" value="1011003"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="190">
    <int name="SN" value="10000192"/>
    <int name="ItemId" value="1042039"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="191">
    <int name="SN" value="10000193"/>
    <int name="ItemId" value="1042040"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="192">
    <int name="SN" value="10000194"/>
    <int name="ItemId" value="1042041"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="193">
    <int name="SN" value="10000195"/>
    <int name="ItemId" value="1061127"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="194">
    <int name="SN" value="10000196"/>
    <int name="ItemId" value="1061128"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="1"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="195">
    <int name="SN" value="10000197"/>
    <int name="ItemId" value="1702010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="196">
    <int name="SN" value="10000198"/>
    <int name="ItemId" value="1702011"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="197">
    <int name="SN" value="10000199"/>
    <int name="ItemId" value="1112213"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="5"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="198">
    <int name="SN" value="10000200"/>
    <int name="ItemId" value="5050002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="199">
    <int name="SN" value="10000201"/>
    <int name="ItemId" value="1000018"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="200">
    <int name="SN" value="10000202"/>
    <int name="ItemId" value="1001023"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="201">
    <int name="SN" value="10000203"/>
    <int name="ItemId" value="1012006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="202">
    <int name="SN" value="10000204"/>
    <int name="ItemId" value="1050109"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="203">
    <int name="SN" value="10000205"/>
    <int name="ItemId" value="1050110"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="204">
    <int name="SN" value="10000206"/>
    <int name="ItemId" value="1051108"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="205">
    <int name="SN" value="10000207"/>
    <int name="ItemId" value="1051109"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="206">
    <int name="SN" value="10000208"/>
    <int name="ItemId" value="1040130"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="207">
    <int name="SN" value="10000209"/>
    <int name="ItemId" value="1040131"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="208">
    <int name="SN" value="10000210"/>
    <int name="ItemId" value="1062039"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="209">
    <int name="SN" value="10000211"/>
    <int name="ItemId" value="1072240"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="210">
    <int name="SN" value="10000212"/>
    <int name="ItemId" value="1702016"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="211">
    <int name="SN" value="10000213"/>
    <int name="ItemId" value="5120006"/>
    <int name="Count" value="11"/>
    <int name="Price" value="3000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="212">
    <int name="SN" value="10000214"/>
    <int name="ItemId" value="5000002"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="213">
    <int name="SN" value="10000215"/>
    <int name="ItemId" value="5000005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="0"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
    <int name="PbCash" value="30"/>
    <int name="PbPoint" value="30"/>
    <int name="PbGift" value="30"/>
  </imgdir>
  <imgdir name="214">
    <int name="SN" value="10000216"/>
    <int name="ItemId" value="1802004"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="215">
    <int name="SN" value="10000217"/>
    <int name="ItemId" value="1802005"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="216">
    <int name="SN" value="10000218"/>
    <int name="ItemId" value="1002197"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="217">
    <int name="SN" value="10000219"/>
    <int name="ItemId" value="1002199"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="218">
    <int name="SN" value="10000220"/>
    <int name="ItemId" value="1002196"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="219">
    <int name="SN" value="10000221"/>
    <int name="ItemId" value="1002198"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="220">
    <int name="SN" value="10000222"/>
    <int name="ItemId" value="1051110"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="221">
    <int name="SN" value="10000223"/>
    <int name="ItemId" value="1051111"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="7"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="222">
    <int name="SN" value="10000224"/>
    <int name="ItemId" value="1042009"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="5"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="223">
    <int name="SN" value="10000225"/>
    <int name="ItemId" value="1042010"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="5"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="224">
    <int name="SN" value="10000226"/>
    <int name="ItemId" value="1042008"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="5"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="225">
    <int name="SN" value="10000227"/>
    <int name="ItemId" value="1042011"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="5"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="226">
    <int name="SN" value="10000228"/>
    <int name="ItemId" value="1040129"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="227">
    <int name="SN" value="10000229"/>
    <int name="ItemId" value="1062012"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="4"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="228">
    <int name="SN" value="10000230"/>
    <int name="ItemId" value="1062013"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="4"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="229">
    <int name="SN" value="10000231"/>
    <int name="ItemId" value="1062011"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="4"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="230">
    <int name="SN" value="10000232"/>
    <int name="ItemId" value="1062014"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="4"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="231">
    <int name="SN" value="10000233"/>
    <int name="ItemId" value="1062040"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="232">
    <int name="SN" value="10000234"/>
    <int name="ItemId" value="1072099"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="3"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="233">
    <int name="SN" value="10000235"/>
    <int name="ItemId" value="1072100"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="3"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="234">
    <int name="SN" value="10000236"/>
    <int name="ItemId" value="1072098"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="3"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="235">
    <int name="SN" value="10000237"/>
    <int name="ItemId" value="1082057"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="2"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="236">
    <int name="SN" value="10000238"/>
    <int name="ItemId" value="1082058"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2400"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="2"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="237">
    <int name="SN" value="10000239"/>
    <int name="ItemId" value="5040000"/>
    <int name="Count" value="6"/>
    <int name="Price" value="2500"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="238">
    <int name="SN" value="10000240"/>
    <int name="ItemId" value="1002260"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="239">
    <int name="SN" value="10000241"/>
    <int name="ItemId" value="1002263"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="240">
    <int name="SN" value="10000242"/>
    <int name="ItemId" value="1032029"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="241">
    <int name="SN" value="10000243"/>
    <int name="ItemId" value="1042046"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4900"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="242">
    <int name="SN" value="10000244"/>
    <int name="ItemId" value="1062041"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="243">
    <int name="SN" value="10000245"/>
    <int name="ItemId" value="1072235"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1800"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="244">
    <int name="SN" value="10000246"/>
    <int name="ItemId" value="1102020"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="245">
    <int name="SN" value="10000247"/>
    <int name="ItemId" value="5160006"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="8"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="246">
    <int name="SN" value="10000248"/>
    <int name="ItemId" value="5150000"/>
    <int name="Count" value="3"/>
    <int name="Price" value="5000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="247">
    <int name="SN" value="10000249"/>
    <int name="ItemId" value="5150004"/>
    <int name="Count" value="3"/>
    <int name="Price" value="5000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="248">
    <int name="SN" value="10000250"/>
    <int name="ItemId" value="5150006"/>
    <int name="Count" value="3"/>
    <int name="Price" value="5000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="249">
    <int name="SN" value="10000251"/>
    <int name="ItemId" value="5150002"/>
    <int name="Count" value="3"/>
    <int name="Price" value="5000"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="6"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="250">
    <int name="SN" value="10000252"/>
    <int name="ItemId" value="1002261"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="251">
    <int name="SN" value="10000253"/>
    <int name="ItemId" value="1002262"/>
    <int name="Count" value="1"/>
    <int name="Price" value="4300"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="252">
    <int name="SN" value="10000254"/>
    <int name="ItemId" value="1041001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="253">
    <int name="SN" value="10000255"/>
    <int name="ItemId" value="1041009"/>
    <int name="Count" value="1"/>
    <int name="Price" value="3700"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="254">
    <int name="SN" value="10000256"/>
    <int name="ItemId" value="1041071"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1200"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="255">
    <int name="SN" value="10000257"/>
    <int name="ItemId" value="1041070"/>
    <int name="Count" value="1"/>
    <int name="Price" value="1200"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="256">
    <int name="SN" value="10000258"/>
    <int name="ItemId" value="1061001"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
  <imgdir name="257">
    <int name="SN" value="10000259"/>
    <int name="ItemId" value="1061007"/>
    <int name="Count" value="1"/>
    <int name="Price" value="2100"/>
    <int name="Period" value="90"/>
    <int name="Priority" value="9"/>
    <int name="Gender" value="2"/>
    <int name="OnSale" value="0"/>
  </imgdir>
</imgdir>
`

func Identity[M any](m M) M {
	return m
}

func TestReader(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 258 {
		t.Fatalf("len(rmm) = %d, want 258", len(rmm))
	}

	var rm RestModel
	var ok bool

	if rm, ok = rmm[strconv.Itoa(10000256)]; !ok {
		t.Fatalf("rmm[10000256] does not exist.")
	}
	if rm.ItemId != 1041071 {
		t.Fatalf("rm.ItemId = %d, want 1041071", rm.ItemId)
	}
	if rm.Count != 1 {
		t.Fatalf("rm.Count = %d, want 1", rm.Count)
	}
	if rm.Price != 1200 {
		t.Fatalf("rm.Price = %d, want 1200", rm.Price)
	}
	if rm.Period != 90 {
		t.Fatalf("rm.Period = %d, want 90", rm.Period)
	}
	if rm.Priority != 9 {
		t.Fatalf("rm.Priority = %d, want 9", rm.Priority)
	}
	if rm.Gender != 2 {
		t.Fatalf("rm.Gender = %d, want 2", rm.Gender)
	}
	if rm.OnSale {
		t.Fatalf("rm.OnSale = true, want false")
	}
}
